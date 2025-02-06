package cmd

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// createTfcCmd represents the tfc command
var createTfcCmd = &cobra.Command{
	Use:    "create-tfc",
	Short:  "Initialize the HCP Terraform Cloud integration",
	Long:   "Create the initial set of parameters to configure HCP Terraform to talk to Overmind.",
	PreRun: PreRunSetup,
	RunE:   CreateTfc,
}

// getTfcCmd represents the tfc command
var getTfcCmd = &cobra.Command{
	Use:    "get-tfc",
	Short:  "Retrieve the existing parameters for the HCP Terraform Cloud integration",
	Long:   "Retrieve the existing parameters for the HCP Terraform Cloud integration.",
	PreRun: PreRunSetup,
	RunE:   GetTfc,
}

// deleteTfcCmd represents the tfc command
var deleteTfcCmd = &cobra.Command{
	Use:    "delete-tfc",
	Short:  "Delete the HCP Terraform Cloud integration",
	Long:   "This will delete the HCP Terraform Cloud integration and disable all access from HCP Terraform Cloud to Overmind.",
	PreRun: PreRunSetup,
	RunE:   DeleteTfc,
}

func init() {
	integrationsCmd.AddCommand(createTfcCmd)
	integrationsCmd.AddCommand(getTfcCmd)
	integrationsCmd.AddCommand(deleteTfcCmd)

	addAPIFlags(createTfcCmd)
	addAPIFlags(getTfcCmd)
	addAPIFlags(deleteTfcCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tfcCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tfcCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func CreateTfc(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"config:write", "api_keys:write", "changes:write", "explore:read", "request:send", "reverselink:request"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedConfigurationClient(ctx, oi)
	fmt.Println("Creating HCP Terraform Cloud integration")
	params, err := client.CreateHcpConfig(ctx, &connect.Request[sdp.CreateHcpConfigRequest]{
		Msg: &sdp.CreateHcpConfigRequest{
			FinalFrontendRedirect: oi.FrontendUrl.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tfc integration: %w", err)
	}

	fmt.Printf("Please visit %v to authorize the integration\nPress return to continue.\n", params.Msg.GetApiKey().GetAuthorizeURL())

	_, err = fmt.Scanln()
	if err != nil {
		return fmt.Errorf("failed waiting for confirmation: %w", err)
	}

	fmt.Println("You can now create a new Run Task in HCP Terraform with the following parameters:")
	fmt.Println("")
	fmt.Println("Name:              Overmind")
	fmt.Println("Endpoint URL:     ", params.Msg.GetConfig().GetEndpoint())
	fmt.Println("Description:       Overmind provides a risk analysis and change tracking for your Terraform changes with no extra effort.")
	fmt.Println("HMAC Key (secret):", params.Msg.GetConfig().GetSecret())
	fmt.Println("")

	log.WithContext(ctx).Info("created tfc integration")
	return nil
}

func GetTfc(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"config:read"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedConfigurationClient(ctx, oi)
	params, err := client.GetHcpConfig(ctx, &connect.Request[sdp.GetHcpConfigRequest]{})
	var cErr *connect.Error
	if errors.As(err, &cErr) {
		if cErr.Code() == connect.CodeNotFound {
			fmt.Println("HCP Terraform Cloud integration is not enabled. Use `create-tfc` to enable it.")
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("failed to get tfc integration params: %w", err)
	}

	fmt.Println("HCP Terraform Cloud integration found")
	fmt.Println("")
	fmt.Println("Name:              Overmind")
	fmt.Println("Endpoint URL:     ", params.Msg.GetConfig().GetEndpoint())
	fmt.Println("Description:       Overmind provides a risk analysis and change tracking for your Terraform changes with no extra effort.")
	fmt.Println("HMAC Key (secret):", params.Msg.GetConfig().GetSecret())
	fmt.Println("")

	return nil
}

func DeleteTfc(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"config:write", "api_keys:write"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedConfigurationClient(ctx, oi)
	fmt.Println("Deleting HCP Terraform Cloud integration")
	_, err = client.DeleteHcpConfig(ctx, &connect.Request[sdp.DeleteHcpConfigRequest]{})
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to delete tfc integration: %w", err)
	}

	log.WithContext(ctx).Info("deleted tfc integration")
	return nil
}
