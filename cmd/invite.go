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

// inviteCmd represents the invites command
var inviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Manage invites for your Overmind account",
	Long: `This command allows you to manage invitations within your Overmind account. When
a user is invited, they will receive an email with a link they can use to sign
up. Once they sign up, they will be added to your account and will have access
to the same data that you do in Overmind`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all invites",
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
	Use:   "create",
	Short: "Create a new invite",
	Run: func(cmd *cobra.Command, args []string) {
		// Bind flags to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind flags")
		}

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
	Use:   "revoke",
	Short: "Revoke an existing invite",
	Run: func(cmd *cobra.Command, args []string) {
		// Bind flags to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind flags")
		}

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
		log.Error("You must specify an email address to revoke using --email")
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI Revoke Invite", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Authenticate
	ctx, err = ensureToken(ctx, []string{"account:write"})

	if err != nil {
		log.Error(err)
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
		log.Error(err)
		return 1
	}

	log.WithFields(log.Fields{"email": email}).Info("Invite revoked successfully")

	return 0
}

func InvitesCreate(ctx context.Context) int {
	var err error

	emails := viper.GetStringSlice("emails")

	if len(emails) == 0 {
		log.Error("You must specify at least one email address to invite using --emails")
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI Create Invite", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Authenticate
	ctx, err = ensureToken(ctx, []string{"account:write"})

	if err != nil {
		log.Error(err)
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
		log.Error(err)
		return 1
	}

	log.WithFields(log.Fields{"emails": emails}).Info("Invites created successfully")

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
		log.Error(err)
		return 1
	}

	client := AuthenticatedInviteClient(ctx)

	// List all invites
	resp, err := client.ListInvites(ctx, &connect.Request[sdp.ListInvitesRequest]{})

	if err != nil {
		log.Error(err)
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
	inviteCmd.AddCommand(listCmd)

	// create sub-command
	inviteCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().StringSlice("emails", []string{}, "A list of emails to invite")

	// revoke sub-command
	inviteCmd.AddCommand(revokeCmd)
	revokeCmd.PersistentFlags().String("email", "", "The email address to revoke")

	rootCmd.AddCommand(inviteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	inviteCmd.PersistentFlags().String("invite-url", "", "A custom URL for the invites API (optional)")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// inviteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
