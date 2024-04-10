package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	"golang.org/x/oauth2"
)

type tfModel struct {
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
}

func NewTfModel(ctx context.Context, action string) tea.Model {
	return tfModel{
		action: action,

		ctx: ctx,

		tasks: make(map[string]tea.Model),
	}
}

func (m tfModel) Init() tea.Cmd {
	// use the main cli context to not take this time from the main timeout
	m.tasks["00_oi"] = NewInstanceLoaderModel(m.ctx, m.app)
	m.tasks["01_token"] = NewEnsureTokenModel(m.ctx, m.app, m.apiKey, m.requiredScopes)
	m.tasks["02_config"] = NewInitialiseSourcesModel() // wait for taking a ctx until timeout and token are attached

	return tea.Batch(
		waitForCancellation(m.ctx, m.cancel),
		m.tasks["00_oi"].Init(),
		m.tasks["01_token"].Init(),
	)
}

func (m tfModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	batch := []tea.Cmd{}

	// pass all messages to all tasks
	for k, t := range m.tasks {
		tm, cmd := t.Update(msg)
		m.tasks[k] = tm
		if cmd != nil {
			batch = append(batch, cmd)
		}
	}

	// special case the messages that need to be handled at this level
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case fatalError:
		if msg.id == 0 {
			m.fatalError = msg.err.Error()
		}
		return m, tea.Quit
		// return m, nil
	case instanceLoadedMsg:
		m.oi = msg.instance
		// skip irrelevant status messages
		// delete(m.tasks, "00_oi")
	case tokenReceivedMsg:
		return m.tokenChecks(msg.token)
	case tokenStoredMsg:
		return m.tokenChecks(msg.token)
	}

	return m, tea.Batch(batch...)
}

func (m tfModel) tokenChecks(token *oauth2.Token) (tfModel, tea.Cmd) {
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

func (m tfModel) View() string {
	tasks := make([]string, 0, len(m.tasks))
	keys := make([]string, 0, len(m.tasks))
	for k := range m.tasks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tasks = append(tasks, m.tasks[k].View())
	}
	if m.fatalError != "" {
		tasks = append(tasks, fmt.Sprintf("Fatal Error: %v", m.fatalError))
	}
	return strings.Join(tasks, "\n")
}
