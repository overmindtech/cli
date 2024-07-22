package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/aws-source/proc"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/sdp-go/auth"
	stdlibsource "github.com/overmindtech/stdlib-source/sources"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type loadSourcesConfigMsg struct {
	ctx    context.Context
	oi     OvermindInstance
	action string
	token  *oauth2.Token
	tfArgs []string
}

type stdlibSourceInitialisedMsg struct{}
type awsSourceInitialisedMsg struct{}

type sourcesInitialisedMsg struct{}
type sourceInitialisationFailedMsg struct{ err error }

// this tea.Model either fetches the AWS auth config from the ConfigService or
// interrogates the user. Results get stored in the ConfigService. Send a
// loadSourcesConfigMsg to start the process. After the sourcesInitialisedMsg
// the viper config has been updated with the values from the ConfigService and
// the sources have successfully loaded and connected to overmind.
type initialiseSourcesModel struct {
	taskModel

	ctx    context.Context // note that this ctx is not initialized on NewGetConfigModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi     OvermindInstance
	action string
	token  *oauth2.Token

	useManagedSources   bool
	awsSourceRunning    bool
	stdlibSourceRunning bool

	errorHints []string

	width int
}

func NewInitialiseSourcesModel(width int) tea.Model {
	return initialiseSourcesModel{
		taskModel: NewTaskModel("Configuring AWS Access", width),

		errorHints: []string{},
	}
}

func (m initialiseSourcesModel) TaskModel() taskModel {
	return m.taskModel
}

func (m initialiseSourcesModel) Init() tea.Cmd {
	return m.taskModel.Init()
}

func (m initialiseSourcesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi
		m.action = msg.action
		m.token = msg.token

		m.status = taskStatusRunning
		if viper.GetBool("only-use-managed-sources") {
			m.useManagedSources = true
			cmds = append(cmds, func() tea.Msg { return sourcesInitialisedMsg{} })
		} else {
			cmds = append(cmds, m.startStdlibSourceCmd(m.ctx, m.oi, m.token))
			cmds = append(cmds, m.startAwsSourceCmd(m.ctx, m.oi, m.token, msg.tfArgs))
		}
		if os.Getenv("CI") == "" {
			cmds = append(cmds, m.spinner.Tick)
		}
	case stdlibSourceInitialisedMsg:
		m.stdlibSourceRunning = true
		if cmdSpan != nil {
			cmdSpan.AddEvent("stdlib source initialised")
		}
		if m.awsSourceRunning {
			cmds = append(cmds, func() tea.Msg { return sourcesInitialisedMsg{} })
		}
	case awsSourceInitialisedMsg:
		m.awsSourceRunning = true
		if cmdSpan != nil {
			cmdSpan.AddEvent("aws source initialised")
		}
		if m.stdlibSourceRunning {
			cmds = append(cmds, func() tea.Msg { return sourcesInitialisedMsg{} })
		}
	case sourcesInitialisedMsg:
		m.status = taskStatusDone
	case sourceInitialisationFailedMsg:
		m.status = taskStatusError
		m.errorHints = append(m.errorHints, "Error initialising sources")
		cmds = append(cmds, func() tea.Msg {
			// create a fatalError for aborting the CLI and common error detail
			// reporting, but don't pass in the spinner ID, to avoid double
			// reporting in this model's View
			return fatalError{err: fmt.Errorf("failed to initialise sources: %w", msg.err)}
		})
	case otherError:
		if msg.id == m.spinner.ID() {
			m.errorHints = append(m.errorHints, fmt.Sprintf("Note: %v", msg.err))
		}
	case fatalError:
		if msg.id == m.spinner.ID() {
			m.status = taskStatusError
			m.errorHints = append(m.errorHints, fmt.Sprintf("Error: %v", msg.err))
		}
	}

	var taskCmd tea.Cmd
	m.taskModel, taskCmd = m.taskModel.Update(msg)
	cmds = append(cmds, taskCmd)

	return m, tea.Batch(cmds...)
}

func (m initialiseSourcesModel) View() string {
	bits := []string{m.taskModel.View()}
	for _, hint := range m.errorHints {
		bits = append(bits, wrap(fmt.Sprintf("  %v %v", RenderErr(), hint), m.width, 2))
	}
	if m.useManagedSources {
		bits = append(bits, wrap(fmt.Sprintf("  %v Using managed sources", RenderOk()), m.width, 2))
	} else {
		if m.awsSourceRunning {
			bits = append(bits, wrap(fmt.Sprintf("  %v AWS Source: running", RenderOk()), m.width, 4))
		}
		if m.stdlibSourceRunning {
			bits = append(bits, wrap(fmt.Sprintf("  %v stdlib Source: running", RenderOk()), m.width, 4))
		}
	}
	return strings.Join(bits, "\n")
}

func natsOptions(ctx context.Context, oi OvermindInstance, token *oauth2.Token) auth.NATSOptions {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	natsNamePrefix := "overmind-cli"

	openapiUrl := *oi.ApiUrl
	openapiUrl.Path = "/api"
	tokenClient := auth.NewOAuthTokenClientWithContext(
		ctx,
		openapiUrl.String(),
		"",
		oauth2.StaticTokenSource(token),
	)

	return auth.NATSOptions{
		NumRetries:        3,
		RetryDelay:        1 * time.Second,
		Servers:           []string{oi.NatsUrl.String()},
		ConnectionName:    fmt.Sprintf("%v.%v", natsNamePrefix, hostname),
		ConnectionTimeout: (10 * time.Second), // TODO: Make configurable
		MaxReconnects:     -1,
		ReconnectWait:     1 * time.Second,
		ReconnectJitter:   1 * time.Second,
		TokenClient:       tokenClient,
	}
}

func (m initialiseSourcesModel) startStdlibSourceCmd(ctx context.Context, oi OvermindInstance, token *oauth2.Token) tea.Cmd {
	return func() tea.Msg {
		natsOptions := natsOptions(ctx, oi, token)

		// ignore returned context. Cancellation of sources is handled by the process exiting for now.
		// should sources require more teardown, we'll have to figure something out.

		stdlibEngine, err := stdlibsource.InitializeEngine(natsOptions, 2_000, true)
		if err != nil {
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("failed to initialize stdlib source engine: %w", err)}
		}

		// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		err = stdlibEngine.Start()

		if err != nil {
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("failed to start stdlib source engine: %w", err)}
		}
		return stdlibSourceInitialisedMsg{}
	}
}

func (m initialiseSourcesModel) startAwsSourceCmd(ctx context.Context, oi OvermindInstance, token *oauth2.Token, tfArgs []string) tea.Cmd {
	return func() tea.Msg {
		varFiles := []string{}
		vars := []string{}

		for _, arg := range tfArgs {
			if strings.HasPrefix(arg, "-var-file=") {
				varFiles = append(varFiles, strings.TrimPrefix(arg, "-var-file="))
			} else if strings.HasPrefix(arg, "-var=") {
				vars = append(vars, strings.TrimPrefix(arg, "-var="))
			} else if strings.HasPrefix(arg, "--var-file=") {
				varFiles = append(varFiles, strings.TrimPrefix(arg, "--var-file="))
			} else if strings.HasPrefix(arg, "--var=") {
				vars = append(vars, strings.TrimPrefix(arg, "--var="))
			}
		}

		tfEval, err := tfutils.LoadEvalContext(varFiles, vars, os.Environ())
		if err != nil {
			return sourceInitialisationFailedMsg{fmt.Errorf("failed to load variables from the environment: %w", err)}
		}

		providers, err := tfutils.ParseAWSProviders(".", tfEval)
		if err != nil {
			return sourceInitialisationFailedMsg{fmt.Errorf("failed to parse providers: %w", err)}
		}
		configs := []aws.Config{}

		for _, p := range providers {
			if p.Error != nil {
				return sourceInitialisationFailedMsg{fmt.Errorf("error when parsing provider: %w", p.Error)}
			}
			c, err := tfutils.ConfigFromProvider(ctx, *p.Provider)
			if err != nil {
				return sourceInitialisationFailedMsg{fmt.Errorf("error when converting provider to config: %w", err)}
			}
			configs = append(configs, c)
		}

		natsOptions := natsOptions(ctx, oi, token)

		awsEngine, err := proc.InitializeAwsSourceEngine(ctx, natsOptions, 2_000, configs...)
		if err != nil {
			return sourceInitialisationFailedMsg{fmt.Errorf("failed to initialize AWS source engine: %w", err)}
		}

		// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		err = awsEngine.Start()
		if err != nil {
			return sourceInitialisationFailedMsg{fmt.Errorf("failed to start AWS source engine: %w", err)}
		}
		return awsSourceInitialisedMsg{}
	}
}
