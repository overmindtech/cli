package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type initialiseSourcesMsg struct {
	ctx    context.Context //note that this ctx is not initialized on NewGetConfigModel to instead get a modified context through the startGetConfigMsg that has a timeout and cancelFunction configured
	oi     OvermindInstance
	action string
	token  *oauth2.Token
}

type sourcesInitialisedMsg struct{}

// this tea.Model either fetches the AWS auth config from the ConfigService or
// interrogates the user. Results get stored in the ConfigService. Send a
// initialiseSourcesMsg to start the process. After the sourcesInitialisedMsg the viper
// config has been updated with the values from the ConfigService.
type initialiseSourcesModel struct {
	taskModel

	ctx    context.Context
	oi     OvermindInstance
	action string
	token  *oauth2.Token

	errors []string
}

func NewInitialiseSourcesModel() tea.Model {
	return initialiseSourcesModel{
		taskModel: NewTaskModel("Configuring AWS Access"),

		errors: []string{},
	}
}

func (m initialiseSourcesModel) Init() tea.Cmd {
	return m.taskModel.Init()
}

func (m initialiseSourcesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initialiseSourcesMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi
		m.action = msg.action
		m.token = msg.token

		m.status = taskStatusRunning
		return m, tea.Batch(
			m.initialiseSourcesCmd(),
			m.spinner.Tick,
		)
	case otherError:
		if msg.id == m.spinner.ID() {
			m.errors = append(m.errors, fmt.Sprintf("Note: %v", msg.err))
		}
		return m, nil
	case fatalError:
		if msg.id == m.spinner.ID() {
			m.status = taskStatusError
			m.title = fmt.Sprintf("Error while configuring AWS Access: %v", msg.err)
		}
		return m, nil
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		return m, taskCmd
	}
}

func (m initialiseSourcesModel) View() string {
	view := m.taskModel.View()
	if len(m.errors) > 0 {
		view = fmt.Sprintf("%v\n%v\n", view, strings.Join(m.errors, "\n"))
	}
	return view
}

func (m initialiseSourcesModel) initialiseSourcesCmd() tea.Cmd {
	ctx := m.ctx

	return func() tea.Msg {
		configClient := AuthenticatedConfigClient(ctx, m.oi)
		cfgValue, err := configClient.GetConfig(ctx, &connect.Request[sdp.GetConfigRequest]{
			Msg: &sdp.GetConfigRequest{
				Key: fmt.Sprintf("cli %v", m.action),
			},
		})
		if err != nil {
			var cErr *connect.Error
			if !errors.As(err, &cErr) || cErr.Code() != connect.CodeNotFound {
				return fatalError{id: m.spinner.ID(), err: fmt.Errorf("failed to get stored config: %w", err)}
			}
		}
		if cfgValue != nil {
			viper.SetConfigType("json")
			err = viper.MergeConfig(bytes.NewBuffer([]byte(cfgValue.Msg.GetValue())))
			if err != nil {
				return fatalError{id: m.spinner.ID(), err: fmt.Errorf("failed to merge stored config: %w", err)}
			}
		}

		// TODO: convert this to a huh.Form using the bubbletea example from the huh repo
		// aws_config := viper.GetString("aws-config")
		// aws_profile := viper.GetString("aws-profile")
		// if aws_config == "" {
		// 	aws_config = "aborted"
		// 	options := []huh.Option[string]{}
		// 	if aws_profile == "" {
		// 		aws_profile = os.Getenv("AWS_PROFILE")
		// 	}
		// 	if aws_profile != "" {
		// 		options = append(options,
		// 			huh.NewOption(fmt.Sprintf("Use $AWS_PROFILE (currently: '%v')", aws_profile), "aws_profile"),
		// 			huh.NewOption("Use a different profile", "profile_input"),
		// 		)
		// 	} else {
		// 		options = append(options,
		// 			huh.NewOption("Use the default settings", "defaults"),
		// 			huh.NewOption("Use an AWS auth profile", "profile_input"),
		// 		)
		// 	}
		// 	// TODO: what URL needs to get opened here?
		// 	// TODO: how to wait for a source to be configured?
		// 	// options = append(options,
		// 	// 	huh.NewOption("Run managed source (opens browser)", "managed"),
		// 	// )
		// 	aws_config_select := huh.NewSelect[string]().
		// 		Title("Choose how to access your AWS account (read-only):").
		// 		Options(options...).
		// 		Value(&aws_config).
		// 		WithAccessible(accessibleMode)
		// 	err = aws_config_select.Run()
		// 	// annoyingly, huh doesn't leave the form on screen - except in
		// 	// accessible mode, so this prints it again so the scrollback looks
		// 	// sensible
		// 	if !accessibleMode {
		// 		fmt.Println(aws_config_select.View())
		// 	}
		// 	if err != nil {
		// 		return func() {}, err
		// 	}
		// }

		return sourcesInitialisedMsg{}
	}
}
