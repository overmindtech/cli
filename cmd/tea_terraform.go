package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type cmdModel struct {
	action string // "plan" or "apply"

	// Context and cancel function from the CmdWrapper. Since bubbletea provides
	// no context handling, we can't follow the usual pattern of keeping the
	// context out of structs.
	ctx    context.Context
	cancel context.CancelFunc

	// configuration
	timeout        time.Duration
	app            string
	apiKey         string
	oi             OvermindInstance // loaded from instanceLoadedMsg
	requiredScopes []string

	// UI state
	tasks               map[string]tea.Model
	terraformHasStarted bool   // remember whether terraform already has started. this is important to do the correct workarounds on errors. See also `skipView()`
	fatalErrorSeen      bool   // remember whether a fatalError has been seen to avoid showing pending tasks
	fatalError          string // this will get set if there's a fatalError coming through that doesn't have a task ID set

	// business logic. This model will implement the actual CLI functionality requested.
	cmd tea.Model
}

type delayQuitMsg struct{}

// fatalError is a wrapper for errors that should abort the running tea.Program.
type fatalError struct {
	id  int
	err error
}

// otherError is a wrapper for errors that should NOT abort the running tea.Program.
type otherError struct {
	id  int
	err error
}

func (m cmdModel) Init() tea.Cmd {
	// use the main cli context to not take this time from the main timeout
	m.tasks["00_oi"] = NewInstanceLoaderModel(m.ctx, m.app)
	m.tasks["01_token"] = NewEnsureTokenModel(m.ctx, m.app, m.apiKey, m.requiredScopes)

	if viper.GetString("ovm-test-fake") != "" {
		// don't init sources on test-fake runs
		// m.tasks["02_config"] = NewInitialiseSourcesModel()
		m.tasks["03_revlink"] = NewRevlinkWarmupModel()
		return tea.Batch(
			waitForCancellation(m.ctx, m.cancel),
			m.tasks["00_oi"].Init(),
			m.tasks["01_token"].Init(),
			// m.tasks["02_config"].Init(),
			func() tea.Msg {
				time.Sleep(3 * time.Second)
				return sourcesInitialisedMsg{}
			},
			m.tasks["03_revlink"].Init(),
			m.cmd.Init(),
		)
	}

	// these wait for taking a ctx until timeout and token are attached
	m.tasks["02_config"] = NewInitialiseSourcesModel()
	m.tasks["03_revlink"] = NewRevlinkWarmupModel()

	return tea.Batch(
		waitForCancellation(m.ctx, m.cancel),
		m.tasks["00_oi"].Init(),
		m.tasks["01_token"].Init(),
		m.tasks["02_config"].Init(),
		m.tasks["03_revlink"].Init(),
		m.cmd.Init(),
	)
}

func (m cmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debugf("cmdModel: Update %T received %#v", msg, msg)

	batch := []tea.Cmd{}

	// special case the messages that need to be handled at this level
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case fatalError:
		log.WithError(msg.err).WithField("msg.id", msg.id).Debug("cmdModel: fatalError received")

		// skipView based on the previous view. While this is not perfect, it's
		// the best we can currently do without taking complete control of
		// terraform command i/o
		if m.terraformHasStarted {
			skipView(m.View())
		}

		m.fatalErrorSeen = true

		// record the fatal error here if it was not from a specific taskModel
		if msg.id == 0 {
			m.fatalError = msg.err.Error()
		}

		return m, tea.Sequence(
			tea.Batch(batch...),
			tea.Quit,
		)

	case instanceLoadedMsg:
		m.oi = msg.instance
		// skip irrelevant status messages
		// delete(m.tasks, "00_oi")

	case tokenAvailableMsg:
		tm, cmd := m.tokenChecks(msg.token)
		batch = append(batch, cmd)
		return tm, tea.Batch(batch...)

	case runPlanNowMsg, runTfApplyMsg:
		m.terraformHasStarted = true

	case runPlanFinishedMsg, tfApplyFinishedMsg:
		// bump screen after terraform ran
		skipView(m.View())

	case delayQuitMsg:
		batch = append(batch, tea.Quit)

	}

	// update the main command
	var cmd tea.Cmd
	m.cmd, cmd = m.cmd.Update(msg)
	if cmd != nil {
		batch = append(batch, cmd)
	}

	// pass all messages to all tasks
	for k, t := range m.tasks {
		tm, cmd := t.Update(msg)
		m.tasks[k] = tm
		if cmd != nil {
			batch = append(batch, cmd)
		}
	}

	return m, tea.Batch(batch...)
}

// skipView scrolls the terminal contents up after ExecCommand() to avoid
// overwriting the output from terraform when rendering the next View(). this
// has to be used here in the cmdModel to catch the entire View() output.
//
// NOTE: this is quite brittle and _requires_ that the View() after terraform
// returned is at least  many lines as the view before ExecCommand(), otherwise
// the difference will get eaten by bubbletea on re-rendering.
//
// TODO: make this hack less ugly
func skipView(view string) {
	lines := strings.Split(view, "\n")
	for range lines {
		fmt.Println()
	}

	// log.Debugf("printed %v lines:", len(lines))
	// log.Debug(lines)
}

func (m cmdModel) tokenChecks(token *oauth2.Token) (cmdModel, tea.Cmd) {
	if viper.GetString("ovm-test-fake") != "" {
		return m, func() tea.Msg {
			return loadSourcesConfigMsg{
				ctx:    m.ctx,
				oi:     m.oi,
				action: m.action,
				token:  token,
			}
		}
	}

	// Check that we actually got the claims we asked for. If you don't have
	// permission auth0 will just not assign those scopes rather than fail
	ok, missing, err := HasScopesFlexible(token, m.requiredScopes)
	if err != nil {
		return m, func() tea.Msg { return fatalError{err: fmt.Errorf("error checking token scopes: %w", err)} }
	}
	if !ok {
		return m, func() tea.Msg {
			return fatalError{err: fmt.Errorf("authenticated successfully, but you don't have the required permission: '%v'", missing)}
		}
	}

	// store the token for later use by sdp-go's auth client. Note that this
	// loses access to the RefreshToken and could be done better by using an
	// oauth2.TokenSource, but this would require more work on updating sdp-go
	// that is currently not scheduled
	m.ctx = context.WithValue(m.ctx, sdp.UserTokenContextKey{}, token.AccessToken)

	// apply the configured timeout to all future operations
	m.ctx, m.cancel = context.WithTimeout(m.ctx, m.timeout)

	// daisy chain the next step. This is a bit of a hack, but it's the easiest
	// for now, and we still need a good idea for a better way. Especially as
	// some of the models require access to viper (for GetConfig/SetConfig) or
	// contortions to store that data somewhere else.
	return m, func() tea.Msg {
		return loadSourcesConfigMsg{
			ctx:    m.ctx,
			oi:     m.oi,
			action: m.action,
			token:  token,
		}
	}
}

func (m cmdModel) View() string {
	// show tasks in key order, skipping pending tasks to keep the ui uncluttered
	allDone := true
	tasks := make([]string, 0, len(m.tasks))
	keys := make([]string, 0, len(m.tasks))
	for k := range m.tasks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t, ok := m.tasks[k].(WithTaskModel)
		if ok {
			if t.TaskModel().status != taskStatusDone {
				allDone = false
			}
			if t.TaskModel().status == taskStatusPending {
				continue
			}
		}
		tasks = append(tasks, m.tasks[k].View())
	}
	if allDone {
		// no need to show setup tasks after they're all done
		tasks = []string{}
	}
	tasks = append(tasks, m.cmd.View())
	if m.fatalError != "" {
		tasks = append(tasks, markdownToString(fmt.Sprintf("> Fatal Error: %v\n", m.fatalError)))
	}
	return strings.Join(tasks, "\n")
}

var applyOnlyArgs = []string{
	"auto-approve",
}

// planArgsFromApplyArgs filters out all apply-specific arguments from arguments
// to `terraform apply`, so that we can run the corresponding `terraform plan`
// command
func planArgsFromApplyArgs(args []string) []string {
	planArgs := []string{}
append:
	for _, arg := range args {
		for _, applyOnlyArg := range applyOnlyArgs {
			if strings.HasPrefix(arg, "-"+applyOnlyArg) {
				continue append
			}
			if strings.HasPrefix(arg, "--"+applyOnlyArg) {
				continue append
			}
		}
		planArgs = append(planArgs, arg)
	}
	return planArgs
}
