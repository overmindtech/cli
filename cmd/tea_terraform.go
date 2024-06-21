package cmd

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getsentry/sentry-go"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	tasks      map[string]tea.Model
	fatalError string // this will get set if there's a fatalError coming through that doesn't have a task ID set

	frozen     bool
	frozenView string // this gets set if the view is frozen, and will be used to render the last view using the cliExecCommand

	hideStartupStatus bool

	// business logic. This model will implement the actual CLI functionality requested.
	cmd tea.Model

	width int
}

type freezeViewMsg struct{}
type unfreezeViewMsg struct{}

type hideStartupStatusMsg struct{}

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

func (m *cmdModel) Init() tea.Cmd {
	// use the main cli context to not take this time from the main timeout
	m.tasks["00_oi"] = NewInstanceLoaderModel(m.ctx, m.app, m.width)
	m.tasks["01_token"] = NewEnsureTokenModel(m.ctx, m.app, m.apiKey, m.requiredScopes, m.width)

	if viper.GetString("ovm-test-fake") != "" {
		// don't init sources on test-fake runs
		// m.tasks["02_config"] = NewInitialiseSourcesModel()
		return tea.Batch(
			m.tasks["00_oi"].Init(),
			m.tasks["01_token"].Init(),
			// m.tasks["02_config"].Init(),
			func() tea.Msg {
				time.Sleep(3 * time.Second)
				return sourcesInitialisedMsg{}
			},
			m.cmd.Init(),
		)
	}

	// these wait for taking a ctx until timeout and token are attached
	m.tasks["02_config"] = NewInitialiseSourcesModel(m.width)

	return tea.Batch(
		m.tasks["00_oi"].Init(),
		m.tasks["01_token"].Init(),
		m.tasks["02_config"].Init(),
		m.cmd.Init(),
	)
}

func (m *cmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	lastMsgType := fmt.Sprintf("%T", msg)
	if lastMsgType != "spinner.TickMsg" {
		log.Debugf("cmdModel: Update %v received %#v", lastMsgType, msg)
		if cmdSpan != nil &&
			strings.HasPrefix(lastMsgType, "cmd.") &&
			!slices.Contains(
				[]string{"cmd.delayQuitMsg", "cmd.fatalError", "cmd.otherError"},
				lastMsgType,
			) {
			cmdSpan.SetAttributes(attribute.String("ovm.cli.lastMsgType", lastMsgType))
		}
	}

	cmds := []tea.Cmd{}

	// special case the messages that need to be handled at this level
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if cmdSpan != nil {
				cmdSpan.SetAttributes(attribute.Bool("ovm.cli.aborted", true))
			}
			return m, tea.Quit
		}
	case freezeViewMsg:
		m.frozenView = m.View()
		m.frozen = true
	case unfreezeViewMsg:
		m.frozen = false
		m.frozenView = ""
	case hideStartupStatusMsg:
		m.hideStartupStatus = true

	case fatalError:
		log.WithError(msg.err).WithField("msg.id", msg.id).Debug("cmdModel: fatalError received")
		span := trace.SpanFromContext(m.ctx)
		span.RecordError(msg.err)
		span.SetAttributes(
			attribute.Bool("ovm.cli.fatalError", true),
			attribute.Int("ovm.cli.fatalError.id", msg.id),
		)
		sentry.CaptureException(msg.err)

		// record the fatal error here, to repeat it at the end of the process
		m.fatalError = msg.err.Error()

		cmds = append(cmds, func() tea.Msg { return delayQuitMsg{} })

	case instanceLoadedMsg:
		m.oi = msg.instance
		// skip irrelevant status messages
		// delete(m.tasks, "00_oi")

	case tokenAvailableMsg:
		var cmd tea.Cmd
		cmd = m.tokenChecks(msg.token)
		cmds = append(cmds, cmd)

	case delayQuitMsg:
		cmds = append(cmds, tea.Quit)

	}

	// update the main command
	var cmd tea.Cmd
	m.cmd, cmd = m.cmd.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// pass all messages to all tasks
	for k, t := range m.tasks {
		tm, cmd := t.Update(msg)
		m.tasks[k] = tm
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *cmdModel) tokenChecks(token *oauth2.Token) tea.Cmd {
	if viper.GetString("ovm-test-fake") != "" {
		return func() tea.Msg {
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
		return func() tea.Msg { return fatalError{err: fmt.Errorf("error checking token scopes: %w", err)} }
	}
	if !ok {
		return func() tea.Msg {
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
	return func() tea.Msg {
		tok, err := josejwt.ParseSigned(token.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			return fatalError{err: fmt.Errorf("received invalid token: %w", err)}
		}
		out := josejwt.Claims{}
		customClaims := sdp.CustomClaims{}
		err = tok.UnsafeClaimsWithoutVerification(&out, &customClaims)
		if err != nil {
			return fatalError{err: fmt.Errorf("received unparsable token: %w", err)}
		}

		if cmdSpan != nil {
			cmdSpan.SetAttributes(
				attribute.Bool("ovm.cli.authenticated", true),
				attribute.String("ovm.cli.accountName", customClaims.AccountName),
				attribute.String("ovm.cli.userId", out.Subject),
			)
		}

		return loadSourcesConfigMsg{
			ctx:    m.ctx,
			oi:     m.oi,
			action: m.action,
			token:  token,
		}
	}
}

func (m cmdModel) View() string {
	if m.frozen {
		return ""
	}
	bits := []string{}

	if !m.hideStartupStatus {
		// show tasks in key order, skipping pending bits to keep the ui uncluttered
		keys := make([]string, 0, len(m.tasks))
		for k := range m.tasks {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t, ok := m.tasks[k].(WithTaskModel)
			if ok {
				if t.TaskModel().status == taskStatusPending {
					continue
				}
			}
			bits = append(bits, m.tasks[k].View())
		}
	}

	bits = append(bits, m.cmd.View())
	if m.fatalError != "" {
		md := markdownToString(m.width, fmt.Sprintf("> Fatal Error: %v\n", m.fatalError))
		md, _ = strings.CutPrefix(md, "\n")
		md, _ = strings.CutSuffix(md, "\n")
		bits = append(bits, md)
	}
	bits = slices.DeleteFunc(bits, func(s string) bool {
		return s == "" || s == "\n"
	})
	return strings.Join(bits, "\n")
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
