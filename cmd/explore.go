package cmd

import (
	"context"
	"fmt"
	"os"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/overmindtech/aws-source/proc"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/discovery"
	"github.com/overmindtech/pterm"
	stdlibsource "github.com/overmindtech/stdlib-source/sources"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// exploreCmd represents the explore command
var exploreCmd = &cobra.Command{
	Use:    "explore",
	Short:  "Run local sources for using in the Explore page",
	Long:   `Run sources locally using terraform's configured authorization to provide data when using https://app.overmind.tech/explore.`,
	PreRun: PreRunSetup,
	RunE:   Explore,

	// SilenceErrors: false,
}

// StartLocalSources runs the local sources using local auth tokens for use by
// any query or request during the runtime of the CLI. for proper cleanup,
// execute the returned function. The method returns once the sources are
// started. Progress is reported into the provided multi printer.
func StartLocalSources(ctx context.Context, oi OvermindInstance, token *oauth2.Token, tfArgs []string, multi *pterm.MultiPrinter) (func(), error) {
	var err error

	stdlibSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting stdlib source engine")
	awsSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting AWS source engine")

	natsOptions := natsOptions(ctx, oi, token)

	p := pool.NewWithResults[*discovery.Engine]().WithErrors()

	p.Go(func() (*discovery.Engine, error) {
		stdlibEngine, err := stdlibsource.InitializeEngine(natsOptions, 2_000, true)
		if err != nil {
			stdlibSpinner.Fail("Failed to initialize stdlib source engine")
			return nil, fmt.Errorf("failed to initialize stdlib source engine: %w", err)
		}

		// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		err = stdlibEngine.Start()
		if err != nil {
			stdlibSpinner.Fail("Failed to start stdlib source engine")
			return nil, fmt.Errorf("failed to start stdlib source engine: %w", err)
		}
		stdlibSpinner.Success("Stdlib source engine started")
		return stdlibEngine, nil
	})

	p.Go(func() (*discovery.Engine, error) {
		tfEval, err := tfutils.LoadEvalContext(tfArgs, os.Environ())
		if err != nil {
			awsSpinner.Fail("Failed to load variables from the environment")
			return nil, fmt.Errorf("failed to load variables from the environment: %w", err)
		}

		providers, err := tfutils.ParseAWSProviders(".", tfEval)
		if err != nil {
			awsSpinner.Fail("Failed to parse providers")
			return nil, fmt.Errorf("failed to parse providers: %w", err)
		}
		configs := []aws.Config{}

		for _, p := range providers {
			if p.Error != nil {
				// skip providers that had errors. This allows us to use
				// providers we _could_ detect, while still failing if there is
				// a true syntax error and no providers are available at all.
				continue
			}
			c, err := tfutils.ConfigFromProvider(ctx, *p.Provider)
			if err != nil {
				awsSpinner.Fail("Error when converting provider to config")
				return nil, fmt.Errorf("error when converting provider to config: %w", err)
			}
			configs = append(configs, c)
		}

		awsEngine, err := proc.InitializeAwsSourceEngine(ctx, natsOptions, 2_000, configs...)
		if err != nil {
			awsSpinner.Fail("Failed to initialize AWS source engine")
			return nil, fmt.Errorf("failed to initialize AWS source engine: %w", err)
		}

		// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		err = awsEngine.Start()
		if err != nil {
			awsSpinner.Fail("Failed to start AWS source engine")
			return nil, fmt.Errorf("failed to start AWS source engine: %w", err)
		}

		awsSpinner.Success("AWS source engine started")
		return awsEngine, nil
	})

	engines, err := p.Wait()
	if err != nil {
		return func() {}, fmt.Errorf("error starting sources: %w", err)
	}

	return func() {
		for _, e := range engines {
			err := e.Stop()
			if err != nil {
				log.WithError(err).Error("failed to stop engine")
			}
		}
	}, nil
}

func Explore(cmd *cobra.Command, args []string) error {
	PTermSetup()

	ctx := cmd.Context()

	multi := pterm.DefaultMultiPrinter

	_, _ = multi.Start()

	ctx, oi, token, err := login(ctx, cmd, []string{"request:receive"}, multi.NewWriter())
	if err != nil {
		return err
	}

	cleanup, err := StartLocalSources(ctx, oi, token, args, &multi)
	if err != nil {
		return err
	}
	defer cleanup()

	_, _ = multi.Stop()

	pterm.Println()
	pterm.Println(fmt.Sprintf("Explore your infrastructure graph at %v/explore", oi.FrontendUrl))
	pterm.Println()
	pterm.Success.Println("Press any key to stop the sources")
	err = keyboard.Listen(func(keyInfo keys.Key) (stop bool, err error) {
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("error reading keyboard input: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(exploreCmd)

	addAPIFlags(exploreCmd)
}
