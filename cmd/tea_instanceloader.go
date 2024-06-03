package cmd

import (
	"context"
	"fmt"
	"net/url"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

type instanceLoadedMsg struct {
	instance OvermindInstance
}

type instanceLoaderModel struct {
	taskModel
	ctx context.Context
	app string
}

func NewInstanceLoaderModel(ctx context.Context, app string) tea.Model {
	result := instanceLoaderModel{
		taskModel: NewTaskModel("Connecting to Overmind"),
		ctx:       ctx,
		app:       app,
	}
	result.status = taskStatusRunning
	return result
}

func (m instanceLoaderModel) TaskModel() taskModel {
	return m.taskModel
}

func (m instanceLoaderModel) Init() tea.Cmd {
	return tea.Batch(
		m.taskModel.Init(),
		newOvermindInstanceCmd(m.ctx, m.app),
	)
}

func (m instanceLoaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg.(type) {
	case instanceLoadedMsg:
		m.status = taskStatusDone
		m.title = "Connected to Overmind"
	}

	var cmd tea.Cmd
	m.taskModel, cmd = m.taskModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func newOvermindInstanceCmd(ctx context.Context, app string) tea.Cmd {
	if viper.GetString("ovm-test-fake") != "" {
		mustParse := func(u string) *url.URL {
			result, err := url.Parse(u)
			if err != nil {
				panic(err)
			}
			return result
		}

		return func() tea.Msg {
			return instanceLoadedMsg{instance: OvermindInstance{
				FrontendUrl: mustParse("http://localhost:3000"),
				ApiUrl:      mustParse("https://api.example.com"),
				NatsUrl:     mustParse("https://nats.example.com"),
				Audience:    "https://aud.example.com",
			}}
		}
	}
	return func() tea.Msg {
		instance, err := NewOvermindInstance(ctx, app)
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to get instance data from app: %w", err)}
		}

		return instanceLoadedMsg{instance}
	}
}
