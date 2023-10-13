package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bufbuild/connect-go"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list-invites",
	Short: "List all invites",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `list-invites` flags")
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

		exitcode := InvitesList(ctx)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create-invite",
	Short: "Create a new invite",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `create-invite` flags")
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

		exitcode := InvitesCreate(ctx)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

// revokeCmd represents the revoke command
var revokeCmd = &cobra.Command{
	Use:   "revoke-invites",
	Short: "Revoke an existing invite",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `revoke-invites` flags")
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

		exitcode := InvitesRevoke(ctx)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func InvitesRevoke(ctx context.Context) int {
	var err error
	email := viper.GetString("email")

	if email == "" {
		log.WithContext(ctx).Error("You must specify an email address to revoke using --email")
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI Revoke Invite", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Authenticate
	ctx, err = ensureToken(ctx, []string{"account:write"})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to ensure token")
		return 1
	}

	client := AuthenticatedInviteClient(ctx)

	// Create the invite
	_, err = client.RevokeInvite(ctx, &connect.Request[sdp.RevokeInviteRequest]{
		Msg: &sdp.RevokeInviteRequest{
			Email: email,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to revoke invite")
		return 1
	}

	log.WithContext(ctx).WithFields(log.Fields{"email": email}).Info("Invite revoked successfully")

	return 0
}

func InvitesCreate(ctx context.Context) int {
	var err error

	emails := viper.GetStringSlice("emails")
	if len(emails) == 0 {
		log.WithContext(ctx).Error("You must specify at least one email address to invite using --emails")
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI Create Invite", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Authenticate
	ctx, err = ensureToken(ctx, []string{"account:write"})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to ensure token")
		return 1
	}

	client := AuthenticatedInviteClient(ctx)

	// Create the invite
	_, err = client.CreateInvite(ctx, &connect.Request[sdp.CreateInviteRequest]{
		Msg: &sdp.CreateInviteRequest{
			Emails: emails,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to create invite")
		return 1
	}

	log.WithContext(ctx).WithFields(log.Fields{"emails": emails}).Info("Invites created successfully")

	return 0
}

func InvitesList(ctx context.Context) int {
	var err error

	ctx, span := tracing.Tracer().Start(ctx, "CLI List Invites", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Authenticate
	ctx, err = ensureToken(ctx, []string{"account:read"})
	if err != nil {
		log.WithError(err).Error("failed to ensure token")
		return 1
	}

	client := AuthenticatedInviteClient(ctx)

	// List all invites
	resp, err := client.ListInvites(ctx, &connect.Request[sdp.ListInvitesRequest]{})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to list invites")
		return 1
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Email", "Status"})

	for _, invite := range resp.Msg.Invites {
		t.AppendRow(table.Row{invite.Email, invite.Status.String()})
	}

	t.Render()

	return 0
}

func init() {
	// list sub-command
	rootCmd.AddCommand(listCmd)
	listCmd.PersistentFlags().String("invite-url", "", "A custom URL for the invites API (optional)")

	// create sub-command
	rootCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().String("invite-url", "", "A custom URL for the invites API (optional)")
	createCmd.PersistentFlags().StringSlice("emails", []string{}, "A list of emails to invite")

	// revoke sub-command
	rootCmd.AddCommand(revokeCmd)
	revokeCmd.PersistentFlags().String("invite-url", "", "A custom URL for the invites API (optional)")
	revokeCmd.PersistentFlags().String("email", "", "The email address to revoke")
}
