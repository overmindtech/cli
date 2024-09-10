package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func PTermSetup() {
	pterm.Success.Prefix.Text = "✔︎"

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
}

func StartSources(ctx context.Context, cmd *cobra.Command, args []string) (context.Context, OvermindInstance, *oauth2.Token, func(), error) {
	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()

	ctx, oi, token, err := login(ctx, cmd, []string{"explore:read", "changes:write", "config:write", "request:receive"}, multi.NewWriter())
	if err != nil {
		return ctx, OvermindInstance{}, nil, nil, err
	}

	cleanup, err := StartLocalSources(ctx, oi, token, args, multi)
	if err != nil {
		return ctx, OvermindInstance{}, nil, nil, err
	}

	return ctx, oi, token, cleanup, nil
}

// start revlink warmup in the background
func RunRevlinkWarmup(ctx context.Context, oi OvermindInstance, postPlanPrinter *atomic.Pointer[pterm.MultiPrinter], args []string) *pool.ErrorPool {
	p := pool.New().WithErrors()
	p.Go(func() error {
		ctx, span := tracing.Tracer().Start(ctx, "revlink warmup")
		defer span.End()

		client := AuthenticatedManagementClient(ctx, oi)
		stream, err := client.RevlinkWarmup(ctx, &connect.Request[sdp.RevlinkWarmupRequest]{
			Msg: &sdp.RevlinkWarmupRequest{},
		})
		if err != nil {
			return fmt.Errorf("error warming up revlink: %w", err)
		}

		// this will get set once the terminal is available
		var spinner *pterm.SpinnerPrinter
		for stream.Receive() {
			msg := stream.Msg()

			if spinner == nil {
				multi := postPlanPrinter.Load()
				if multi != nil {
					// start the spinner in the background, now that a multi
					// printer is available
					spinner, _ = pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Discovering and linking all resources")
				}
			}

			// only update the spinner if we have access to the terminal
			if spinner != nil {
				items := msg.GetItems()
				edges := msg.GetEdges()
				if items+edges > 0 {
					spinner.UpdateText(fmt.Sprintf("Discovering and linking all resources: %v (%v items, %v edges)", msg.GetStatus(), items, edges))
				} else {
					spinner.UpdateText(fmt.Sprintf("Discovering and linking all resources: %v", msg.GetStatus()))
				}
			}
		}

		err = stream.Err()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			if spinner != nil {
				spinner.Fail(fmt.Sprintf("Error warming up revlink: %v", err))
			}
			return fmt.Errorf("error warming up revlink: %w", err)
		}

		if spinner != nil {
			spinner.Success("Discovered and linked all resources")
		}

		return nil
	})

	return p
}

func RunPlan(ctx context.Context, args []string) error {
	c := exec.CommandContext(ctx, "terraform", args...) // nolint:gosec // this is a user-provided command, let them do their thing

	// remove go's default process cancel behaviour, so that terraform has a
	// chance to gracefully shutdown when ^C is pressed. Otherwise the
	// process would get killed immediately and leave locks lingering behind
	c.Cancel = func() error {
		return nil
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	_, span := tracing.Tracer().Start(ctx, "terraform plan")
	defer span.End()

	log.WithField("args", c.Args).Debug("running terraform plan")

	err := c.Run()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to run terraform plan: %w", err)
	}

	return nil
}

func RunApply(ctx context.Context, args []string) error {
	c := exec.CommandContext(ctx, "terraform", args...) // nolint:gosec // this is a user-provided command, let them do their thing

	// remove go's default process cancel behaviour, so that terraform has a
	// chance to gracefully shutdown when ^C is pressed. Otherwise the
	// process would get killed immediately and leave locks lingering behind
	c.Cancel = func() error {
		return nil
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	_, span := tracing.Tracer().Start(ctx, "terraform apply")
	defer span.End()

	log.WithField("args", c.Args).Debug("running terraform apply")

	err := c.Run()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to run terraform apply: %w", err)
	}

	return nil
}
