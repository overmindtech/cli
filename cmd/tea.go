package cmd

// this file contains a bunch of general helpers for building commands based on the bubbletea framework

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type OvermindCommandHandler func(ctx context.Context, args []string, oi OvermindInstance, token *oauth2.Token) error

// viperGetApp fetches and validates the configured app url
func viperGetApp(ctx context.Context) (string, error) {
	app := viper.GetString("app")

	// Check to see if the URL is secure
	parsed, err := url.Parse(app)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --app")
		return "", fmt.Errorf("error parsing --app: %w", err)
	}

	if !(parsed.Scheme == "wss" || parsed.Scheme == "https" || parsed.Hostname() == "localhost") {
		return "", fmt.Errorf("target URL (%v) is insecure", parsed)
	}
	return app, nil
}

type FinalReportingModel interface {
	FinalReport() string
}

func CmdWrapper(action string, requiredScopes []string, commandModel func(args []string, parent *cmdModel, width int) tea.Model) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		// set up a context for the command
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		cmdName := fmt.Sprintf("CLI %v", cmd.CommandPath())
		defer tracing.LogRecoverToExit(ctx, cmdName)

		// ensure that only error messages are printed to the console,
		// disrupting bubbletea rendering (and potentially getting overwritten).
		// Otherwise, when TEABUG is set, log to a file.
		if len(os.Getenv("TEABUG")) > 0 {
			f, err := tea.LogToFile("teabug.log", "debug")
			if err != nil {
				fmt.Println("fatal:", err)
				os.Exit(1)
			}
			// leave the log file open until the very last moment, so we capture everything
			// defer f.Close()
			log.SetOutput(f)
			formatter := new(log.TextFormatter)
			formatter.DisableTimestamp = false
			log.SetFormatter(formatter)
			viper.Set("log", "trace")
			log.SetLevel(log.TraceLevel)
		} else {
			// avoid log messages from sources and others to interrupt bubbletea rendering
			viper.Set("log", "fatal")
			log.SetLevel(log.FatalLevel)
		}

		// wrap the rest of the function in a closure to allow for cleaner error handling and deferring.
		err := func() error {
			timeout, err := time.ParseDuration(viper.GetString("timeout"))
			if err != nil {
				return flagError{usage: fmt.Sprintf("invalid --timeout value '%v'\n\n%v", viper.GetString("timeout"), cmd.UsageString())}
			}

			app, err := viperGetApp(ctx)
			if err != nil {
				return err
			}

			m := cmdModel{
				action:         action,
				ctx:            ctx,
				cancel:         cancel,
				timeout:        timeout,
				app:            app,
				requiredScopes: requiredScopes,
				args:           args,
				apiKey:         viper.GetString("api-key"),
				tasks:          map[string]tea.Model{},
			}
			m.cmd = commandModel(args, &m, m.width)

			options := []tea.ProgramOption{}
			if os.Getenv("CI") != "" {
				// See https://github.com/charmbracelet/bubbletea/issues/761#issuecomment-1625863769
				options = append(options, tea.WithInput(nil))
			}
			p := tea.NewProgram(&m, options...)
			result, err := p.Run()
			if err != nil {
				return fmt.Errorf("could not start program: %w", err)
			}

			cmd, ok := result.(*cmdModel)
			if ok {
				frm, ok := cmd.cmd.(FinalReportingModel)
				if ok {
					fmt.Println(frm.FinalReport())
				}
			}

			return nil
		}()
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error running command")
			// don't forget that os.Exit() does not wait for telemetry to be flushed
			if cmdSpan != nil {
				cmdSpan.End()
			}
			tracing.ShutdownTracer()
			os.Exit(1)
		}
	}
}
