package cmd

import (
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:    "list-invites",
	Short:  "List all invites",
	PreRun: PreRunSetup,
	RunE:   InvitesList,
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:    "create-invite",
	Short:  "Create a new invite",
	PreRun: PreRunSetup,
	RunE:   InvitesCreate,
}

// revokeCmd represents the revoke command
var revokeCmd = &cobra.Command{
	Use:    "revoke-invites",
	Short:  "Revoke an existing invite",
	PreRun: PreRunSetup,
	RunE:   InvitesRevoke,
}

func InvitesRevoke(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var err error
	email := viper.GetString("email")
	if email == "" {
		log.WithContext(ctx).Error("You must specify an email address to revoke using --email")
		return flagError{usage: fmt.Sprintf("You must specify an email address to revoke using --email\n\n%v", cmd.UsageString())}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"account:write"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedInviteClient(ctx, oi)

	// Create the invite
	_, err = client.RevokeInvite(ctx, &connect.Request[sdp.RevokeInviteRequest]{
		Msg: &sdp.RevokeInviteRequest{
			Email: email,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  log.Fields{"email": email},
			message: "failed to revoke invite",
		}
	}

	log.WithContext(ctx).WithFields(log.Fields{"email": email}).Info("Invite revoked successfully")

	return nil
}

func InvitesCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	emails := viper.GetStringSlice("emails")
	if len(emails) == 0 {
		return flagError{usage: fmt.Sprintf("You must specify at least one email address to invite using --emails\n\n%v", cmd.UsageString())}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"account:write"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedInviteClient(ctx, oi)

	// Create the invite
	_, err = client.CreateInvite(ctx, &connect.Request[sdp.CreateInviteRequest]{
		Msg: &sdp.CreateInviteRequest{
			Emails: emails,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  log.Fields{"emails": emails},
			message: "failed to create invite",
		}
	}

	log.WithContext(ctx).WithFields(log.Fields{"emails": emails}).Info("Invites created successfully")

	return nil
}

func InvitesList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"account:read"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedInviteClient(ctx, oi)

	// List all invites
	resp, err := client.ListInvites(ctx, &connect.Request[sdp.ListInvitesRequest]{})
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to list invites",
		}
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Email", "Status"})

	for _, invite := range resp.Msg.GetInvites() {
		t.AppendRow(table.Row{invite.GetEmail(), invite.GetStatus().String()})
	}

	t.Render()

	return nil
}

func init() {
	// list sub-command
	invitesCmd.AddCommand(listCmd)

	// create sub-command
	invitesCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().StringSlice("emails", []string{}, "A list of emails to invite")

	// revoke sub-command
	invitesCmd.AddCommand(revokeCmd)
	revokeCmd.PersistentFlags().String("email", "", "The email address to revoke")
}
