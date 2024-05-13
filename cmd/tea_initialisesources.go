package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/overmindtech/sdp-go"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type loadSourcesConfigMsg struct {
	ctx    context.Context
	oi     OvermindInstance
	action string
	token  *oauth2.Token
}

type askForAwsConfigMsg struct{}
type configStoredMsg struct{}
type sourcesInitialisedMsg struct{}

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

	awsConfigForm        *huh.Form // is set if the user needs to be interrogated about their aws_config
	awsConfigFormDone    bool      // gets set to true once the form result has been processed
	profileInputForm     *huh.Form // is set if the user needs to be interrogated about their profile_input
	profileInputFormDone bool      // gets set to true once the form result has been processed

	configStored bool

	awsSourceRunning    bool
	stdlibSourceRunning bool

	errors []string
}

func NewInitialiseSourcesModel() tea.Model {
	return initialiseSourcesModel{
		taskModel: NewTaskModel("Configuring AWS Access"),

		errors: []string{},
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
	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi
		m.action = msg.action
		m.token = msg.token

		m.status = taskStatusRunning
		cmds = append(cmds, m.loadSourcesConfigCmd)
		cmds = append(cmds, m.spinner.Tick)
	case askForAwsConfigMsg:
		// load the config that was injected above. If it's not there, prompt the user.
		aws_config := viper.GetString("aws-config")
		aws_profile := viper.GetString("aws-profile")

		if aws_config == "" || viper.GetBool("reset-stored-config") {
			aws_config = "aborted"
			options := []huh.Option[string]{}
			aws_profile_env := os.Getenv("AWS_PROFILE")
			// TODO: add a "managed" option
			if aws_profile == aws_profile_env && aws_profile != "" {
				// the value of $AWS_PROFILE was not overridden on the commandline
				options = append(options,
					huh.NewOption("Use the default settings", "defaults"),
					huh.NewOption(fmt.Sprintf("Use $AWS_PROFILE (currently: '%v')", aws_profile_env), "aws_profile"),
					huh.NewOption("Select a different AWS auth profile", "profile_input"),
				)
			} else {
				if aws_profile != "" {
					// used --aws-profile on the command line, with a value different from $AWS_PROFILE
					options = append(options,
						huh.NewOption("Use the default settings", "defaults"),
						huh.NewOption(fmt.Sprintf("Use the selected AWS profile (currently: '%v')", aws_profile), "aws_profile"),
						huh.NewOption("Select a different AWS auth profile", "profile_input"),
					)
				} else {
					options = append(options,
						huh.NewOption("Use the default settings", "defaults"),
						huh.NewOption("Select an AWS auth profile", "profile_input"),
					)
				}
			}

			// TODO: what URL needs to get opened here?
			// TODO: how to wait for a source to be configured?
			// options = append(options,
			// 	huh.NewOption("Run managed source (opens browser)", "managed"),
			// )

			selector := huh.NewSelect[string]().
				Key("aws-config").
				Title("Choose how to access your AWS account (read-only):").
				Options(options...)
			m.awsConfigForm = huh.NewForm(huh.NewGroup(selector))
			cmds = append(cmds, selector.Focus())
			selector.Skip()
		} else {
			m.awsConfigFormDone = true

			if aws_config == "profile_input" && aws_profile == "" {
				input := huh.NewInput().
					Key("aws-profile").
					Title("Input the name of the AWS profile to use:")
				m.profileInputForm = huh.NewForm(
					huh.NewGroup(input),
				)
				cmds = append(cmds, input.Focus())
			} else {
				cmds = append(cmds, m.storeConfigCmd(aws_config, aws_profile))
				cmds = append(cmds, m.startSourcesCmd(aws_config, aws_profile))
			}
		}
	case configStoredMsg:
		m.configStored = true
	case sourcesInitialisedMsg:
		m.awsSourceRunning = true
		m.stdlibSourceRunning = true
		m.status = taskStatusDone
	case otherError:
		if msg.id == m.spinner.ID() {
			m.errors = append(m.errors, fmt.Sprintf("Note: %v", msg.err))
		}
	case fatalError:
		if msg.id == m.spinner.ID() {
			m.status = taskStatusError
			m.title = markdownToString(fmt.Sprintf("> error while configuring AWS access: %v", msg.err))
		}
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		cmds = append(cmds, taskCmd)
	}

	// process the form if it is not yet done
	if m.awsConfigForm != nil && !m.awsConfigFormDone {
		switch m.awsConfigForm.State {
		case huh.StateAborted:
			m.awsConfigFormDone = true
			// well, shucks
			return m, tea.Quit
		case huh.StateNormal:
			// pass on messages while the form is active
			form, cmd := m.awsConfigForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.awsConfigForm = f
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case huh.StateCompleted:
			m.awsConfigFormDone = true

			// store the result locally
			aws_config := m.awsConfigForm.GetString("aws-config")
			viper.Set("aws-config", aws_config)

			// ask the next question if required
			if aws_config == "profile_input" {
				input := huh.NewInput().
					Key("aws-profile").
					Title("Input the name of the AWS profile to use:")
				m.profileInputForm = huh.NewForm(
					huh.NewGroup(input),
				)
				cmds = append(cmds, input.Focus())
			} else {
				// no input required; skip the next question
				m.profileInputFormDone = true
				aws_profile := viper.GetString("aws-profile")
				cmds = append(cmds, m.storeConfigCmd(aws_config, aws_profile))
				cmds = append(cmds, m.startSourcesCmd(aws_config, aws_profile))
			}
		}
	}

	// process the form if it exists and is not yet done
	if m.profileInputForm != nil && !m.profileInputFormDone {
		switch m.profileInputForm.State {
		case huh.StateAborted:
			m.profileInputFormDone = true
			// well, shucks
			return m, tea.Quit
		case huh.StateNormal:
			// pass on messages while the form is active
			form, cmd := m.profileInputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.profileInputForm = f
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case huh.StateCompleted:
			m.profileInputFormDone = true
			// store the result
			viper.Set("aws-profile", m.profileInputForm.GetString("aws-profile"))
			cmds = append(cmds, m.storeConfigCmd(viper.GetString("aws-config"), viper.GetString("aws-profile")))
			cmds = append(cmds, m.startSourcesCmd(viper.GetString("aws-config"), viper.GetString("aws-profile")))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m initialiseSourcesModel) View() string {
	view := m.taskModel.View()
	if m.configStored {
		view += " (config stored)"
	}
	if len(m.errors) > 0 {
		view += fmt.Sprintf("\n%v\n", strings.Join(m.errors, "\n"))
	}
	if m.awsConfigForm != nil {
		view += fmt.Sprintf("\n%v", m.awsConfigForm.View())
	}
	if m.profileInputForm != nil {
		view += fmt.Sprintf("\n%v", m.profileInputForm.View())
	}
	if m.awsSourceRunning {
		view += "\n✅ AWS Source: running"
	}
	if m.stdlibSourceRunning {
		view += "\n✅ stdlib Source: running"
	}
	return view
}

func (m initialiseSourcesModel) loadSourcesConfigCmd() tea.Msg {
	ctx := m.ctx
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

	return askForAwsConfigMsg{}
}

func (m initialiseSourcesModel) storeConfigCmd(aws_config, aws_profile string) tea.Cmd {
	return func() tea.Msg {
		ctx := m.ctx
		configClient := AuthenticatedConfigClient(ctx, m.oi)

		jsonBuf, err := json.Marshal(terraformStoredConfig{
			Config:  aws_config,
			Profile: aws_profile,
		})
		if err != nil {
			return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to marshal config: %w", err)}
		}
		_, err = configClient.SetConfig(ctx, &connect.Request[sdp.SetConfigRequest]{
			Msg: &sdp.SetConfigRequest{
				Key:   fmt.Sprintf("cli %v", m.action),
				Value: string(jsonBuf),
			},
		})
		if err != nil {
			return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to upload config: %w", err)}
		}

		return configStoredMsg{}
	}
}
func (m initialiseSourcesModel) startSourcesCmd(aws_config, aws_profile string) tea.Cmd {
	return func() tea.Msg {
		// ignore returned context. Cancellation of sources is handled by the process exiting for now.
		// should sources require more teardown, we'll have to figure something out.
		_, err := InitializeSources(m.ctx, m.oi, aws_config, aws_profile, m.token)
		if err != nil {
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("failed to initialise sources: %w", err)}
		}
		return sourcesInitialisedMsg{}
	}
}
