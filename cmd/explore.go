package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/google/uuid"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/aws-source/proc"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	gcpproc "github.com/overmindtech/cli/sources/gcp/proc"
	stdlibSource "github.com/overmindtech/cli/stdlib-source/adapters"
	"github.com/overmindtech/cli/tracing"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// exploreCmd represents the explore command
var exploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Run local sources for using in the Explore page",
	Long: `Run sources locally using terraform's configured authorization to provide data when using https://app.overmind.tech/explore.

The CLI automatically discovers and uses:
- AWS providers from your Terraform configuration
- GCP providers from your Terraform configuration (google and google-beta)
- Falls back to default cloud provider credentials if no Terraform providers are found

For GCP, ensure you have appropriate permissions (roles/browser or equivalent) to access project metadata.`,
	PreRun: PreRunSetup,
	RunE:   Explore,

	// SilenceErrors: false,
}

// StartLocalSources runs the local sources using local auth tokens for use by
// any query or request during the runtime of the CLI. for proper cleanup,
// execute the returned function. The method returns once the sources are
// started. Progress is reported into the provided multi printer.
func StartLocalSources(ctx context.Context, oi sdp.OvermindInstance, token *oauth2.Token, tfArgs []string, failOverToAws bool) (func(), error) {
	var err error

	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()
	stdlibSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting stdlib source engine")
	awsSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting AWS source engine")
	gcpSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting GCP source engine")
	statusArea := pterm.DefaultParagraph.WithWriter(multi.NewWriter())

	natsOptions := natsOptions(ctx, oi, token)
	heartbeatOptions := heartbeatOptions(oi, token)

	hostname, err := os.Hostname()
	if err != nil {
		return func() {}, fmt.Errorf("failed to get hostname: %w", err)
	}

	p := pool.NewWithResults[[]*discovery.Engine]().WithErrors()

	p.Go(func() ([]*discovery.Engine, error) { //nolint:contextcheck // todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		ec := discovery.EngineConfig{
			Version:               fmt.Sprintf("cli-%v", tracing.Version()),
			EngineType:            "cli-stdlib",
			SourceName:            fmt.Sprintf("stdlib-source-%v", hostname),
			SourceUUID:            uuid.New(),
			App:                   oi.ApiUrl.Host,
			ApiKey:                token.AccessToken,
			NATSOptions:           &natsOptions,
			MaxParallelExecutions: 2_000,
			HeartbeatOptions:      heartbeatOptions,
		}
		stdlibEngine, err := stdlibSource.InitializeEngine(
			&ec,
			true,
		)
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
		return []*discovery.Engine{stdlibEngine}, nil
	})

	p.Go(func() ([]*discovery.Engine, error) {
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
				statusArea.Println(fmt.Sprintf("Skipping AWS provider in %s with %s.", p.FilePath, p.Error.Error()))
				continue
			}
			c, err := tfutils.ConfigFromProvider(ctx, *p.Provider)
			if err != nil {
				awsSpinner.Fail("Error when converting provider to config")
				return nil, fmt.Errorf("error when converting provider to config: %w", err)
			}
			credentials, _ := c.Credentials.Retrieve(ctx)
			statusArea.Println(fmt.Sprintf("Using AWS provider in %s with %s.", p.FilePath, credentials.Source))
			configs = append(configs, c)
		}
		if len(configs) == 0 && failOverToAws {
			userConfig, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				awsSpinner.Fail("Failed to load default AWS config")
				return nil, fmt.Errorf("failed to load default AWS config: %w", err)
			}
			statusArea.Println("Using default AWS CLI config. No AWS terraform providers found.")
			configs = append(configs, userConfig)
		}
		ec := discovery.EngineConfig{
			EngineType:            "cli-aws",
			Version:               fmt.Sprintf("cli-%v", tracing.Version()),
			SourceName:            fmt.Sprintf("aws-source-%v", hostname),
			SourceUUID:            uuid.New(),
			App:                   oi.ApiUrl.Host,
			ApiKey:                token.AccessToken,
			MaxParallelExecutions: 2_000,
			NATSOptions:           &natsOptions,
			HeartbeatOptions:      heartbeatOptions,
		}
		awsEngine, err := proc.InitializeAwsSourceEngine(
			ctx,
			&ec,
			1, // Don't retry as we want the user to get notified immediately
			configs...,
		)
		if err != nil {
			if os.Getenv("AWS_PROFILE") == "" {
				// look for the AWS_PROFILE env var and suggest setting it
				awsSpinner.Fail("Failed to initialize AWS source engine. Consider setting AWS_PROFILE to use the default AWS CLI profile.")
			} else {
				awsSpinner.Fail("Failed to initialize AWS source engine")
			}
			return nil, fmt.Errorf("failed to initialize AWS source engine: %w", err)
		}

		err = awsEngine.Start() //nolint:contextcheck // todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		if err != nil {
			awsSpinner.Fail("Failed to start AWS source engine")
			return nil, fmt.Errorf("failed to start AWS source engine: %w", err)
		}

		awsSpinner.Success("AWS source engine started")
		return []*discovery.Engine{awsEngine}, nil
	})

	p.Go(func() ([]*discovery.Engine, error) {
		// Parse GCP providers from Terraform configuration
		tfEval, err := tfutils.LoadEvalContext(tfArgs, os.Environ())
		if err != nil {
			gcpSpinner.Fail("Failed to load variables from the environment for GCP")
			return nil, fmt.Errorf("failed to load variables from the environment for GCP: %w", err)
		}

		gcpProviders, err := tfutils.ParseGCPProviders(".", tfEval)
		if err != nil {
			gcpSpinner.Fail("Failed to parse GCP providers")
			return nil, fmt.Errorf("failed to parse GCP providers: %w", err)
		}

		// Process GCP providers and extract configurations
		gcpConfigs := []*gcpproc.GCPConfig{}

		for _, p := range gcpProviders {
			if p.Error != nil {
				statusArea.Println(fmt.Sprintf("Skipping GCP provider in %s: %s", p.FilePath, p.Error.Error()))
				continue
			}

			config, err := tfutils.ConfigFromGCPProvider(*p.Provider)
			if err != nil {
				statusArea.Println(fmt.Sprintf("Error configuring GCP provider in %s: %s", p.FilePath, err.Error()))
				continue
			}

			gcpConfigs = append(gcpConfigs, &gcpproc.GCPConfig{
				ProjectID: config.ProjectID,
				Regions:   config.Regions,
				Zones:     config.Zones,
			})

			aliasInfo := ""
			if config.Alias != "" {
				aliasInfo = fmt.Sprintf(" (alias: %s)", config.Alias)
			}
			statusArea.Println(fmt.Sprintf("Using GCP provider in %s with project %s%s", p.FilePath, config.ProjectID, aliasInfo))
		}

		// Fallback to default GCP config if no terraform providers found
		if len(gcpConfigs) == 0 {
			statusArea.Println("No GCP terraform providers found. Attempting to use default GCP credentials.")
			// Try to use Application Default Credentials by passing nil config
			gcpConfigs = append(gcpConfigs, nil)
		}

		// Start multiple GCP engines for each configuration
		gcpEngines := []*discovery.Engine{}
		for i, gcpConfig := range gcpConfigs {
			engineSuffix := ""
			if len(gcpConfigs) > 1 {
				engineSuffix = fmt.Sprintf("-%d", i)
			}

			ec := discovery.EngineConfig{
				EngineType:            "cli-gcp",
				Version:               fmt.Sprintf("cli-%v", tracing.Version()),
				SourceName:            fmt.Sprintf("gcp-source-%v%s", hostname, engineSuffix),
				SourceUUID:            uuid.New(),
				App:                   oi.ApiUrl.Host,
				ApiKey:                token.AccessToken,
				MaxParallelExecutions: 2_000,
				NATSOptions:           &natsOptions,
				HeartbeatOptions:      heartbeatOptions,
			}

			gcpEngine, err := gcpproc.Initialize(ctx, &ec, gcpConfig)
			if err != nil {
				if gcpConfig == nil {
					// Default config failed
					statusArea.Println(fmt.Sprintf("Failed to initialize GCP source with default credentials: %s", err.Error()))
				} else {
					statusArea.Println(fmt.Sprintf("Failed to initialize GCP source for project %s: %s", gcpConfig.ProjectID, err.Error()))
				}
				continue // Skip this engine but continue with others
			}

			err = gcpEngine.Start() //nolint:contextcheck
			if err != nil {
				if gcpConfig == nil {
					statusArea.Println(fmt.Sprintf("Failed to start GCP source with default credentials: %s", err.Error()))
				} else {
					statusArea.Println(fmt.Sprintf("Failed to start GCP source for project %s: %s", gcpConfig.ProjectID, err.Error()))
				}
				continue // Skip this engine but continue with others
			}

			gcpEngines = append(gcpEngines, gcpEngine)
		}

		if len(gcpEngines) == 0 {
			gcpSpinner.Fail("Failed to initialize any GCP source engines")
			return nil, nil // skip GCP if there are no valid configurations
		}

		if len(gcpEngines) == 1 {
			gcpSpinner.Success("GCP source engine started")
		} else {
			gcpSpinner.Success(fmt.Sprintf("%d GCP source engines started", len(gcpEngines)))
		}

		return gcpEngines, nil
	})

	engines, err := p.Wait()
	if err != nil {
		return func() {}, fmt.Errorf("error starting sources: %w", err)
	}

	return func() {
		for _, e := range slices.Concat(engines...) {
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
	_, _ = multi.Start() // multi-printer controls the lifecycle of screen output, it needs to be stopped before printing anything else
	defer func() {
		_, _ = multi.Stop()
	}()
	ctx, oi, token, err := login(ctx, cmd, []string{"request:receive", "api:read"}, multi.NewWriter())
	_, _ = multi.Stop()
	if err != nil {
		return err
	}

	cleanup, err := StartLocalSources(ctx, oi, token, args, true)
	if err != nil {
		return err
	}
	defer cleanup()

	exploreURL := fmt.Sprintf("%v/explore", oi.FrontendUrl)
	_ = browser.OpenURL(exploreURL) // ignore error, we can't do anything about it

	pterm.Println()
	pterm.Println(fmt.Sprintf("Explore your infrastructure graph at %s", exploreURL))
	pterm.Println()
	pterm.Success.Println("Press Ctrl+C to stop the locally running sources")
	err = keyboard.Listen(func(keyInfo keys.Key) (stop bool, err error) {
		if keyInfo.Code == keys.CtrlC {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error reading keyboard input: %w", err)
	}

	// This spinner will spin forever as the command shuts down as this could
	// take a couple of seconds and we want the user to know it's doing
	// something
	_, _ = pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Shutting down")

	return nil
}

func init() {
	rootCmd.AddCommand(exploreCmd)

	addAPIFlags(exploreCmd)
}
