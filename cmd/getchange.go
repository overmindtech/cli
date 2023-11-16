package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

//go:embed comment.md
var commentTemplate string

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:   "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short: "Displays the contents of a change.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-change` flags")
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

		exitcode := GetChange(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI GetChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, []string{"changes:read"})
	if err != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	lf := log.Fields{}
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]), true)
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	client := AuthenticatedChangesClient(ctx)
	changeRes, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"change-url": viper.GetString("change-url"),
		}).Error("failed to get change")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid":        uuid.UUID(changeRes.Msg.Change.Metadata.UUID),
		"change-created":     changeRes.Msg.Change.Metadata.CreatedAt.AsTime(),
		"change-status":      changeRes.Msg.Change.Metadata.Status.String(),
		"change-name":        changeRes.Msg.Change.Properties.Title,
		"change-description": changeRes.Msg.Change.Properties.Description,
	}).Info("found change")

	// diffRes, err := client.GetDiff(ctx, &connect.Request[sdp.GetDiffRequest]{
	// 	Msg: &sdp.GetDiffRequest{
	// 		ChangeUUID: changeUuid[:],
	// 	},
	// })
	// if err != nil {
	// 	log.WithContext(ctx).WithError(err).WithFields(log.Fields{
	// 		"change-url": viper.GetString("change-url"),
	// 	}).Error("failed to get change diff")
	// 	return 1
	// }
	// log.WithContext(ctx).WithFields(log.Fields{
	// 	"change-uuid": uuid.UUID(changeRes.Msg.Change.Metadata.UUID),
	// }).Info("loaded change diff")

	switch viper.GetString("format") {
	case "json":
		b, err := json.MarshalIndent(changeRes.Msg.Change.ToMap(), "", "  ")
		if err != nil {
			log.WithContext(ctx).WithField("input", fmt.Sprintf("%#v", changeRes.Msg.Change.ToMap())).WithError(err).Error("Error rendering change")
			return 1
		}

		fmt.Println(string(b))
	case "markdown":
		type TemplateItem struct {
			StatusAlt  string
			StatusIcon string
			Type       string
			Title      string
			Diff       string
		}
		type TemplateRisk struct {
			SeverityAlt  string
			SeverityIcon string
			SeverityText string
			Title        string
			Description  string
		}
		type TemplateData struct {
			ChangeUrl       string
			ExpectedChanges []TemplateItem
			UnmappedChanges []TemplateItem
			BlastItems      int
			BlastEdges      int
			Risks           []TemplateRisk
		}
		data := TemplateData{
			ChangeUrl: fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), changeUuid.String()),
			ExpectedChanges: []TemplateItem{
				{
					StatusAlt:  "updated",
					StatusIcon: "https://github.com/overmindtech/ovm-cli/assets/8799341/db3e59a9-a560-4ea9-b38b-854d88ab2cd6",
					Type:       "Deployment",
					Title:      "api-server",
					Diff:       "  Once again a diff here",
				},
			},
			UnmappedChanges: []TemplateItem{
				{
					StatusAlt:  "created",
					StatusIcon: "https://github.com/overmindtech/ovm-cli/assets/8799341/2fc6cb63-9ee1-4e15-91ea-b234a92edddb",
					Type:       "auth0_action",
					Title:      "add_to_crm",
					Diff:       "  This should be diff with everything",
				}, {
					StatusAlt:  "updated",
					StatusIcon: "https://github.com/overmindtech/ovm-cli/assets/8799341/db3e59a9-a560-4ea9-b38b-854d88ab2cd6",
					Type:       "auth0_action",
					Title:      "create_account",
					Diff: `  code: |
  const { createPromiseClient } = require( "@connectrpc/connect")
  const { createConnectTransport } = require( "@connectrpc/connect-node")
- const { Auth0Support,Auth0CreateUserRequest } = require('@overmindtech/sdp-js')
+ const { Auth0Support} = require('@overmindtech/sdp-js')
  const ClientOAuth2 = require('client-oauth2')
  const transport = createConnectTransport({
  ---
  api.accessToken.setCustomClaim('https://api.overmind.tech/account-name', event.user.app_metadata.account_name)
  // wake up all sources
- const res = await client.keepaliveSources(
+ await client.keepaliveSources(
  {
    account: event.user.app_metadata.account_name,
  },
`,
				},
			},
			BlastItems: 75,
			BlastEdges: 97,
			Risks: []TemplateRisk{
				{
					SeverityAlt:  "high",
					SeverityIcon: "https://github.com/overmindtech/ovm-cli/assets/8799341/76a34fd0-7699-4636-9a4c-3cdabdf7783e",
					SeverityText: "high",
					Title:        "Impact on Target Groups",
					Description:  `The various target groups including \"944651592624.eu-west-2.elbv2-target-group.k8s-default-nats-4650f3a363\", \"944651592624.eu-west-2.elbv2-target-group.k8s-default-smartloo-fd2416d9f8\", etc., that work alongside the load balancer for traffic routing may be indirectly affected if the security group change causes networking issues. This is especially important if these target groups rely on different ports other than 8080 for health checks or for directing incoming requests to backend services.`,
				},
			},
		}
		tmpl, err := template.New("comment").Parse(commentTemplate)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("error parsing comment template")
			return 1
		}
		err = tmpl.Execute(os.Stdout, data)
		if err != nil {
			log.WithContext(ctx).WithField("input", fmt.Sprintf("%#v", data)).WithError(err).Error("error rendering comment")
			return 1
		}
	}

	return 0
}

func init() {
	rootCmd.AddCommand(getChangeCmd)

	withChangeUuidFlags(getChangeCmd)
	getChangeCmd.PersistentFlags().String("status", "", "The expected status of the change. Use this with --ticket-link. Allowed values: CHANGE_STATUS_UNSPECIFIED, CHANGE_STATUS_DEFINING, CHANGE_STATUS_HAPPENING, CHANGE_STATUS_PROCESSING, CHANGE_STATUS_DONE")

	getChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")
	getChangeCmd.PersistentFlags().String("format", "json", "How to render the change. Possible values: json, markdown")

	getChangeCmd.PersistentFlags().String("timeout", "5m", "How long to wait for responses")
}
