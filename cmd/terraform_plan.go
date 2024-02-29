package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	awssource "github.com/overmindtech/aws-source/cmd"
	"github.com/overmindtech/cli/cmd/datamaps"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/auth"
	stdlibsource "github.com/overmindtech/stdlib-source/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// terraformPlanCmd represents the `terraform plan` command
var terraformPlanCmd = &cobra.Command{
	Use:   "plan [overmind options...] -- [terraform options...]",
	Short: "Runs `terraform plan` and sends the results to Overmind to calculate a blast radius and risks.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `terraform plan` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goroutine to watch for cancellation signals
		go func() {
			select {
			case <-sigs:
				cancel()
			case <-ctx.Done():
			}
		}()

		exitcode := TerraformPlan(ctx, args, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func TerraformPlan(ctx context.Context, args []string, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI TerraformPlan", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	hostname, err := os.Hostname()
	if err != nil {
		log.WithError(err).Fatal("Could not determine hostname for use in NATS connection name")
	}

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get instance data from app")
		return 1
	}

	ctx, token, err := ensureToken(ctx, oi, []string{"changes:write", "request:receive"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	_, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	r := NewTermRenderer()

	// TODO: store this in the api-server and skip questioning the user after the first time
	aws_config := "aborted"
	options := []huh.Option[string]{}
	aws_profile := os.Getenv("AWS_PROFILE")
	if aws_profile != "" {
		options = append(options,
			huh.NewOption(fmt.Sprintf("Use $AWS_PROFILE (currently: '%v')", aws_profile), "aws_profile"),
			huh.NewOption("Use a different profile", "profile_input"),
		)
	} else {
		options = append(options,
			huh.NewOption("Use the default settings", "defaults"),
			huh.NewOption("Use an AWS SSO profile", "profile_input"),
		)
	}
	// TODO: what URL needs to get opened here?
	// TODO: how to wait for a source to be configured?
	// options = append(options,
	// 	huh.NewOption("Run managed source (opens browser)", "managed"),
	// )
	aws_config_select := huh.NewSelect[string]().
		Title("Choose how to access your AWS account (read-only):").
		Options(options...).
		Value(&aws_config).
		WithAccessible(accessibleMode)
	err = aws_config_select.Run()
	// annoyingly, huh doesn't leave the form on screen - except in
	// accessible mode, so this prints it again so the scrollback looks
	// sensible
	if !accessibleMode {
		fmt.Println(aws_config_select.View())
	}
	if err != nil {
		fmt.Printf("Aborting: %v\n", err)
		return 1
	}

	awsAuthConfig := awssource.AwsAuthConfig{
		Regions: []string{"eu-west-1"},
	}

	switch aws_config {
	case "profile_input":
		aws_profile_input := huh.NewInput().
			Title("Input the name of the AWS profile to use:").
			Value(&aws_profile).
			WithAccessible(accessibleMode)
		err = aws_profile_input.Run()
		// annoyingly, huh doesn't leave the form on screen - except in
		// accessible mode, so this prints it again so the scrollback looks
		// sensible
		if !accessibleMode {
			fmt.Println(aws_profile_input.View())
		}
		if err != nil {
			fmt.Printf("Aborting: %v\n", err)
			return 1
		}
		// reset the environment to the requested value
		awsAuthConfig.Strategy = "sso-profile"
		awsAuthConfig.Profile = aws_profile
	case "aws_profile":
		// can continue with the existing config
		awsAuthConfig.Strategy = "sso-profile"
		awsAuthConfig.Profile = aws_profile
	case "defaults":
		// just continue
		awsAuthConfig.Strategy = "defaults"
	case "managed":
		// TODO: not implemented yet
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

	natsOptions := auth.NATSOptions{
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

	awsEngine, err := awssource.InitializeAwsSourceEngine(natsOptions, awsAuthConfig, 2_000)
	if err != nil {
		log.WithError(err).Error("failed to initialize AWS source engine")
		return 1
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = awsEngine.Start()
	if err != nil {
		log.WithError(err).Error("failed to start AWS source engine")
		return 1
	}
	defer func() {
		_ = awsEngine.Stop()
	}()

	stdlibEngine, err := stdlibsource.InitializeStdlibSourceEngine(natsOptions, 2_000, true)
	if err != nil {
		log.WithError(err).Error("failed to initialize stdlib source engine")
		return 1
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = stdlibEngine.Start()
	if err != nil {
		log.WithError(err).Error("failed to start stdlib source engine")
		return 1
	}
	defer func() {
		_ = stdlibEngine.Stop()
	}()

	args = append([]string{"plan"}, args...)
	// -out needs to go last to override whatever the user specified on the command line
	args = append(args, "-out", "overmind.plan")

	prompt := `
* AWS Source: running
* stdlib Source: running

# Planning Changes

Running ` + "`" + `terraform %v` + "`" + `
`

	out, err := r.Render(fmt.Sprintf(prompt, strings.Join(args, " ")))
	if err != nil {
		panic(err)
	}
	fmt.Print(out)

	// TODO: remove this debugging aid
	err = os.Chdir("/workspace/deploy")
	if err != nil {
		panic(err)
	}

	tfPlanCmd := exec.CommandContext(ctx, "terraform", args...)
	tfPlanCmd.Stderr = os.Stderr
	tfPlanCmd.Stdout = os.Stdout
	tfPlanCmd.Stdin = os.Stdin

	err = tfPlanCmd.Run()
	if err != nil {
		log.WithError(err).Error("failed to run terraform plan")
		return 1
	}

	tfPlanJsonCmd := exec.CommandContext(ctx, "terraform", "show", "-json", "overmind.plan")
	tfPlanJsonCmd.Stderr = os.Stderr

	planJson, err := tfPlanJsonCmd.Output()
	if err != nil {
		log.WithError(err).Error("failed to convert terraform plan to JSON")
		return 1
	}

	plannedChanges, err := mappedItemDiffsFromPlan(ctx, planJson, "overmind.plan", lf)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Error parsing terraform plan")
		return 1
	}

	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		h := sha256.New()
		h.Write(planJson)
		ticketLink = fmt.Sprintf("tfplan://{SHA256}%x", h.Sum(nil))
	}
	client := AuthenticatedChangesClient(ctx, oi)
	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, false)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed searching for existing changes")
		return 1
	}

	title := changeTitle(viper.GetString("title"))
	tfPlanOutput := tryLoadText(ctx, viper.GetString("terraform-plan-output"))
	codeChangesOutput := tryLoadText(ctx, viper.GetString("code-changes-diff"))

	if changeUuid == uuid.Nil {
		log.WithContext(ctx).WithFields(lf).Debug("Creating a new change")
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  ticketLink,
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
					RawPlan:     tfPlanOutput,
					CodeChanges: codeChangesOutput,
				},
			},
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to create change")
			return 1
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read change id")
			return 1
		}

		changeUuid = *maybeChangeUuid
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("Created a new change")
	} else {
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Debug("Updating an existing change")

		_, err := client.UpdateChange(ctx, &connect.Request[sdp.UpdateChangeRequest]{
			Msg: &sdp.UpdateChangeRequest{
				UUID: changeUuid[:],
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  ticketLink,
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
					RawPlan:     tfPlanOutput,
					CodeChanges: codeChangesOutput,
				},
			},
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to update change")
			return 1
		}

		log.WithContext(ctx).WithFields(lf).Info("Re-using change")
	}

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: plannedChanges,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to update planned changes")
		return 1
	}

	last_log := time.Now()
	first_log := true
	for resultStream.Receive() {
		msg := resultStream.Msg()

		// log the first message and at most every 250ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.GetState() != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(lf).WithField("msg", msg).Info("Status update")
			last_log = time.Now()
			first_log = false
		}
	}
	if resultStream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(resultStream.Err()).Error("Error streaming results")
		return 1
	}

	changeUrl := *oi.FrontendUrl
	changeUrl.Path = fmt.Sprintf("%v/changes/%v/blast-radius", changeUrl.Path, changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl.String()).Info("Change ready")
	fmt.Println(changeUrl.String())

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to get updated change")
		return 1
	}

	for _, a := range fetchResponse.Msg.GetChange().GetProperties().GetAffectedAppsUUID() {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).WithField("app", a).Error("Received invalid app uuid")
			continue
		}
		appUrl := *oi.FrontendUrl
		appUrl.Path = fmt.Sprintf("%v/apps/%v", appUrl.Path, appUuid)
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"change-url": changeUrl,
			"app":        appUuid,
			"app-url":    appUrl.String(),
		}).Info("Affected app")
	}

	return 0
}

func mappedItemDiffsFromPlan(ctx context.Context, planJson []byte, fileName string, lf log.Fields) ([]*sdp.MappedItemDiff, error) {
	// Check that we haven't been passed a state file
	if isStateFile(planJson) {
		return nil, fmt.Errorf("'%v' appears to be a state file, not a plan file", fileName)
	}

	var plan Plan
	err := json.Unmarshal(planJson, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse '%v': %w", fileName, err)
	}

	plannedChangeGroupsVar := plannedChangeGroups{
		supported:   map[string][]*sdp.MappedItemDiff{},
		unsupported: map[string][]*sdp.MappedItemDiff{},
	}

	// for all managed resources:
	for _, resourceChange := range plan.ResourceChanges {
		if len(resourceChange.Change.Actions) == 0 || resourceChange.Change.Actions[0] == "no-op" || resourceChange.Mode == "data" {
			// skip resources with no changes and data updates
			continue
		}

		itemDiff, err := itemDiffFromResourceChange(resourceChange)
		if err != nil {
			return nil, fmt.Errorf("failed to create item diff for resource change: %w", err)
		}

		awsMappings := datamaps.AwssourceData[resourceChange.Type]
		k8sMappings := datamaps.K8ssourceData[resourceChange.Type]

		mappings := append(awsMappings, k8sMappings...)

		if len(mappings) == 0 {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
			plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: nil, // unmapped item has no mapping query
			})
			continue
		}

		for _, mapData := range mappings {
			var currentResource *Resource

			// Look for the resource in the prior values first, since this is
			// the *previous* state we're like to be able to find it in the
			// actual infra
			if plan.PriorState.Values != nil {
				currentResource = plan.PriorState.Values.RootModule.DigResource(resourceChange.Address)
			}

			// If we didn't find it, look in the planned values
			if currentResource == nil {
				currentResource = plan.PlannedValues.RootModule.DigResource(resourceChange.Address)
			}

			if currentResource == nil {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", resourceChange.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("Skipping resource without values")
				continue
			}

			query, ok := currentResource.AttributeValues.Dig(mapData.QueryField)
			if !ok {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", resourceChange.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("Skipping resource without query field")
				continue
			}

			// Create the map that variables will pull data from
			dataMap := make(map[string]any)

			// Populate resource values
			dataMap["values"] = currentResource.AttributeValues

			if overmindMappingsOutput, ok := plan.PlannedValues.Outputs["overmind_mappings"]; ok {
				configResource := plan.Config.RootModule.DigResource(resourceChange.Address)

				if configResource == nil {
					log.WithContext(ctx).
						WithFields(lf).
						WithField("terraform-address", resourceChange.Address).
						Debug("Skipping provider mapping for resource without config")
				} else {
					// Look up the provider config key in the mappings
					mappings := make(map[string]map[string]string)

					err = json.Unmarshal(overmindMappingsOutput.Value, &mappings)

					if err != nil {
						log.WithContext(ctx).
							WithFields(lf).
							WithField("terraform-address", resourceChange.Address).
							WithError(err).
							Error("Failed to parse overmind_mappings output")
					} else {
						// We need to split out the module section of the name
						// here. If the resource isn't in a module, the
						// ProviderConfigKey will be something like
						// "kubernetes", however if it's in a module it's be
						// something like "module.something:kubernetes"
						providerName := extractProviderNameFromConfigKey(configResource.ProviderConfigKey)
						currentProviderMappings, ok := mappings[providerName]

						if ok {
							log.WithContext(ctx).
								WithFields(lf).
								WithField("terraform-address", resourceChange.Address).
								WithField("provider-config-key", configResource.ProviderConfigKey).
								Debug("Found provider mappings")

							// We have mappings for this provider, so set them
							// in the `provider_mapping` value
							dataMap["provider_mapping"] = currentProviderMappings
						}
					}
				}
			}

			// Interpolate variables in the scope
			scope, err := InterpolateScope(mapData.Scope, dataMap)

			if err != nil {
				log.WithContext(ctx).WithError(err).Debugf("Could not find scope mapping variables %v, adding them will result in better results. Error: ", mapData.Scope)
				scope = "*"
			}

			u := uuid.New()
			newQuery := &sdp.Query{
				Type:               mapData.Type,
				Method:             mapData.Method,
				Query:              fmt.Sprintf("%v", query),
				Scope:              scope,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
				Deadline:           timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			// cleanup item metadata from mapping query
			if itemDiff.GetBefore() != nil {
				itemDiff.Before.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.Before.Scope = newQuery.GetScope()
				}
			}

			// cleanup item metadata from mapping query
			if itemDiff.GetAfter() != nil {
				itemDiff.After.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.After.Scope = newQuery.GetScope()
				}
			}

			plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: newQuery,
			})

			log.WithContext(ctx).WithFields(log.Fields{
				"scope":  newQuery.GetScope(),
				"type":   newQuery.GetType(),
				"query":  newQuery.GetQuery(),
				"method": newQuery.GetMethod().String(),
			}).Debug("Mapped resource to query")
		}
	}

	supported := ""
	numSupported := plannedChangeGroupsVar.NumSupportedChanges()
	if numSupported > 0 {
		supported = Green.Color(fmt.Sprintf("%v supported", numSupported))
	}

	unsupported := ""
	numUnsupported := plannedChangeGroupsVar.NumUnsupportedChanges()
	if numUnsupported > 0 {
		unsupported = Yellow.Color(fmt.Sprintf("%v unsupported", numUnsupported))
	}

	numTotalChanges := numSupported + numUnsupported

	switch numTotalChanges {
	case 0:
		log.WithContext(ctx).Infof("Plan (%v) contained no changing resources.", fileName)
	case 1:
		log.WithContext(ctx).Infof("Plan (%v) contained one changing resource: %v %v", fileName, supported, unsupported)
	default:
		log.WithContext(ctx).Infof("Plan (%v) contained %v changing resources: %v %v", fileName, numTotalChanges, supported, unsupported)
	}

	// Log the types
	for typ, plannedChanges := range plannedChangeGroupsVar.supported {
		log.WithContext(ctx).Infof(Green.Color("  ✓ %v (%v)"), typ, len(plannedChanges))
	}
	for typ, plannedChanges := range plannedChangeGroupsVar.unsupported {
		log.WithContext(ctx).Infof(Yellow.Color("  ✗ %v (%v)"), typ, len(plannedChanges))
	}

	return plannedChangeGroupsVar.MappedItemDiffs(), nil
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
	addChangeUuidFlags(terraformPlanCmd)
}
