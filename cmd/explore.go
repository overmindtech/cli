package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

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
	"github.com/spf13/viper"
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
func StartLocalSources(ctx context.Context, oi sdp.OvermindInstance, token *oauth2.Token, tfArgs []string, failOverToDefaultLoginCfg bool) (func(), error) {
	var err error

	// Default to recursive search unless --no-recursion is set
	tfRecursive := !viper.GetBool("no-recursion")

	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()

	natsOpts := natsOptions(ctx, oi, token)

	hostname, err := os.Hostname()
	if err != nil {
		return func() {}, fmt.Errorf("failed to get hostname: %w", err)
	}

	p := pool.NewWithResults[[]*discovery.Engine]().WithErrors()

	// find all the terraform files
	tfFiles, err := tfutils.FindTerraformFiles(".", tfRecursive)
	if err != nil {
		// we only error if there is a filesystem error, 0 files is handled below
		return nil, err
	}

	// if no terraform files are found, return an error
	if len(tfFiles) == 0 && !failOverToDefaultLoginCfg {
		currentDir, _ := os.Getwd()
		msgLines := []string{
			fmt.Sprintf("No Terraform configuration files found in %s", currentDir),
			"",
			"The Overmind CLI requires access to Terraform configuration files (.tf files) to discover and authenticate with cloud providers. Without Terraform configuration, the CLI cannot determine which cloud resources to interrogate.",
			"",
			"To resolve this issue:",
			"- Ensure you're running the command from a directory containing Terraform files (.tf files)",
			"- Or create Terraform configuration files that define your cloud providers",
			"",
		}
		if !tfRecursive {
			msgLines = append(msgLines, "- Or remove --no-recursion to scan subdirectories for Terraform stacks")
		}
		msgLines = append(msgLines, "For more information about Terraform configuration, visit: https://developer.hashicorp.com/terraform/language")
		return nil, errors.New(strings.Join(msgLines, "\n"))
	}

	stdlibSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting stdlib source engine")
	awsSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting AWS source engine")
	gcpSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting GCP source engine")
	statusArea := pterm.DefaultParagraph.WithWriter(multi.NewWriter())

	foundCloudProvider := false

	p.Go(func() ([]*discovery.Engine, error) { //nolint:contextcheck // todo: pass in context with timeout to abort timely and allow Ctrl-C to work
		ec := discovery.EngineConfig{
			Version:               fmt.Sprintf("cli-%v", tracing.Version()),
			EngineType:            "cli-stdlib",
			SourceName:            fmt.Sprintf("stdlib-source-%v", hostname),
			SourceUUID:            uuid.New(),
			App:                   oi.ApiUrl.Host,
			ApiKey:                token.AccessToken,
			NATSOptions:           &natsOpts,
			MaxParallelExecutions: 2_000,
			HeartbeatOptions:      heartbeatOptions(oi, token),
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

		awsProviders, err := tfutils.ParseAWSProviders(".", tfEval, tfRecursive)
		if err != nil {
			awsSpinner.Fail("Failed to parse AWS providers")
			return nil, fmt.Errorf("failed to parse AWS providers: %w", err)
		}

		if len(awsProviders) == 0 && !failOverToDefaultLoginCfg {
			awsSpinner.Warning("No AWS terraform providers found, skipping AWS source initialization.")
			return nil, nil // skip AWS if there are no awsProviders
		}

		configs := []aws.Config{}
		for _, p := range awsProviders {
			if p.Error != nil {
				// skip providers that had errors. This allows us to use
				// providers we _could_ detect, while still failing if there is
				// a true syntax error and no providers are available at all.
				statusArea.Println(fmt.Sprintf("Skipping AWS provider in %s with %s.", p.FilePath, p.Error.Error()))
				continue
			}
			c, err := tfutils.ConfigFromProvider(ctx, *p.Provider)
			if err != nil {
				awsSpinner.Fail("Error when converting AWS Terraform provider to config: ", err)
				return nil, fmt.Errorf("error when converting AWS Terraform provider to config: %w", err)
			}
			credentials, _ := c.Credentials.Retrieve(ctx)
			aliasInfo := ""
			if p.Provider.Alias != "" {
				aliasInfo = fmt.Sprintf(" (alias: %s)", p.Provider.Alias)
			}
			statusArea.Println(fmt.Sprintf("Using AWS provider %s%s in %s with %s.", p.Provider.Name, aliasInfo, p.FilePath, credentials.Source))
			configs = append(configs, c)
		}
		if len(configs) == 0 && failOverToDefaultLoginCfg {
			statusArea.Println("No AWS terraform providers found. Attempting to use AWS default credentials for configuration.")
			userConfig, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				awsSpinner.Fail("Failed to load default AWS config: ", err)
				return nil, fmt.Errorf("failed to load default AWS config: %w", err)
			}
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
			NATSOptions:           &natsOpts,
			HeartbeatOptions:      heartbeatOptions(oi, token),
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
		foundCloudProvider = true
		return []*discovery.Engine{awsEngine}, nil
	})

	p.Go(func() ([]*discovery.Engine, error) {
		// Parse GCP providers from Terraform configuration
		tfEval, err := tfutils.LoadEvalContext(tfArgs, os.Environ())
		if err != nil {
			gcpSpinner.Fail("Failed to load variables from the environment for GCP")
			return nil, fmt.Errorf("failed to load variables from the environment for GCP: %w", err)
		}

		gcpProviders, err := tfutils.ParseGCPProviders(".", tfEval, tfRecursive)
		if err != nil {
			gcpSpinner.Fail("Failed to parse GCP providers")
			return nil, fmt.Errorf("failed to parse GCP providers: %w", err)
		}

		if len(gcpProviders) == 0 && !failOverToDefaultLoginCfg {
			gcpSpinner.Warning("No GCP terraform providers found, skipping GCP source initialization.")
			return nil, nil // skip GCP if there are no providers
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
				statusArea.Println(fmt.Sprintf("Error configuring GCP provider %s in %s: %s", p.Provider.Name, p.FilePath, err.Error()))
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
			statusArea.Println(fmt.Sprintf("Using GCP provider in %s with project %s%s.", p.FilePath, config.ProjectID, aliasInfo))
		}

		gcpConfigs = unifiedGCPConfigs(gcpConfigs)

		// Fallback to default GCP config if no terraform providers found
		if len(gcpConfigs) == 0 && failOverToDefaultLoginCfg {
			statusArea.Println("No GCP terraform providers found. Attempting to use GCP Application Default Credentials for configuration.")
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
				NATSOptions:           &natsOpts,
				HeartbeatOptions:      heartbeatOptions(oi, token),
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

		foundCloudProvider = true
		return gcpEngines, nil
	})

	engines, err := p.Wait()
	if err != nil {
		return func() {}, fmt.Errorf("error starting sources: %w", err)
	}

	if !foundCloudProvider {
		statusArea.Println(`No cloud providers found in Terraform configuration.

The Overmind CLI requires access to cloud provider configurations to interrogate resources. Without configured providers, the CLI cannot determine which cloud resources to query and as a result calculate a successful blast radius.

To resolve this issue ensure your Terraform configuration files define at least one supported cloud provider (e.g., AWS, GCP)

For more information about configuring cloud providers in Terraform, visit:
- AWS: https://registry.terraform.io/providers/hashicorp/aws/latest/docs
- GCP: https://registry.terraform.io/providers/hashicorp/google/latest/docs`)
	}

	// Return a cleanup function to stop all engines
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
	// flag to opt-out of recursion and only scan the current folder for *.tf files
	exploreCmd.PersistentFlags().Bool("no-recursion", false, "Only scan the current directory for Terraform files (non-recursive).")
}

// unifiedGCPConfigs collates the given GCP configs by project ID.
// If there are multiple configs for the same project ID, the configs are merged.
func unifiedGCPConfigs(gcpConfigs []*gcpproc.GCPConfig) []*gcpproc.GCPConfig {
	unified := make(map[string]*gcpproc.GCPConfig)
	for _, config := range gcpConfigs {
		if _, ok := unified[config.ProjectID]; !ok {
			unified[config.ProjectID] = config
		} else {
			unified[config.ProjectID].Regions = append(unified[config.ProjectID].Regions, config.Regions...)
			unified[config.ProjectID].Zones = append(unified[config.ProjectID].Zones, config.Zones...)
		}
	}

	unifiedConfigs := make([]*gcpproc.GCPConfig, 0, len(unified))
	for _, config := range unified {
		var deDuplicatedRegions []string
		var deDuplicatedZones []string
		for _, region := range config.Regions {
			if !slices.Contains(deDuplicatedRegions, region) {
				deDuplicatedRegions = append(deDuplicatedRegions, region)
			}
		}
		for _, zone := range config.Zones {
			if !slices.Contains(deDuplicatedZones, zone) {
				deDuplicatedZones = append(deDuplicatedZones, zone)
			}
		}
		config.Regions = deDuplicatedRegions
		config.Zones = deDuplicatedZones
		unifiedConfigs = append(unifiedConfigs, config)
	}

	return unifiedConfigs
}
