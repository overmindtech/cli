package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// This file contains re-usable sets of flags that should be used when creating
// commands

// Adds flags for selecting a change by UUID, frontend URL or ticket link
func addChangeUuidFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("change", "", "The frontend URL of the change to get")
	cmd.PersistentFlags().String("ticket-link", "", "Link to the ticket for this change.")
	cmd.PersistentFlags().String("uuid", "", "The UUID of the change that should be displayed.")
	cmd.MarkFlagsMutuallyExclusive("change", "ticket-link", "uuid")
}

// Adds flags that should be present when creating a change
func addChangeCreationFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("title", "", "Short title for this change. If this is not specified, overmind will try to come up with one for you.")
	cmd.PersistentFlags().String("description", "", "Quick description of the change.")
	cmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change. Usually this would be the link to something like the pull request, since the CLI uses this as a unique identifier for the change, meaning that multiple runs with the same ticket link will update the same change.")
	cmd.PersistentFlags().String("owner", "", "The owner of this change.")
	cmd.PersistentFlags().String("repo", "", "The repository URL that this change should be linked to. This will be automatically detected is possible from the Git config or CI environment.")
	cmd.PersistentFlags().String("terraform-plan-output", "", "Filename of cached terraform plan output for this change.")
	cmd.PersistentFlags().String("code-changes-diff", "", "Filename of the code diff of this change.")
	cmd.PersistentFlags().StringSlice("tags", []string{}, "Tags to apply to this change, these should be specified in key=value format. Multiple tags can be specified by repeating the flag or using a comma separated list.")
	cmd.PersistentFlags().StringSlice("labels", []string{}, "Labels to apply to this change, these should be specified in name=color format where color is a hex code (e.g., FF0000 or #FF0000). Multiple labels can be specified by repeating the flag or using a comma separated list.")
}

func parseTagsArgument() (*sdp.EnrichedTags, error) {
	tags := map[string]string{}
	// get into key pair
	for _, tag := range viper.GetStringSlice("tags") {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format: %s", tag)
		}
		tags[parts[0]] = parts[1]
	}
	// put into enriched tags
	enrichedTags := &sdp.EnrichedTags{
		TagValue: make(map[string]*sdp.TagValue),
	}
	for key, value := range tags {
		enrichedTags.TagValue[key] = &sdp.TagValue{
			Value: &sdp.TagValue_UserTagValue{
				UserTagValue: &sdp.UserTagValue{
					Value: value,
				},
			},
		}
	}
	return enrichedTags, nil
}

func parseLabelsArgument() ([]*sdp.Label, error) {
	labels := make([]*sdp.Label, 0)
	for _, label := range viper.GetStringSlice("labels") {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %s (expected name=color)", label)
		}
		if parts[0] == "" {
			return nil, fmt.Errorf("invalid label format: %s (label name cannot be empty)", label)
		}

		// Normalise colour: strip leading # if present, validate, then add # back
		colour := strings.TrimPrefix(parts[1], "#")
		if colour == "" {
			return nil, fmt.Errorf("invalid colour format: %s (colour cannot be empty)", parts[1])
		}

		// Validate it's exactly 6 hex digits
		if len(colour) != 6 {
			return nil, fmt.Errorf("invalid colour format: %s (must be 6 hex digits, got %d)", parts[1], len(colour))
		}

		// Validate all characters are valid hex digits
		if _, err := strconv.ParseUint(colour, 16, 64); err != nil {
			return nil, fmt.Errorf("invalid colour format: %s (must be valid hex digits)", parts[1])
		}

		// Normalise to canonical form: always #rrggbb
		normalisedColour := "#" + strings.ToUpper(colour)

		labels = append(labels, &sdp.Label{
			Name:   parts[0],
			Colour: normalisedColour,
			Type:   sdp.LabelType_LABEL_TYPE_USER,
		})
	}
	return labels, nil
}

// Adds common flags to API commands e.g. timeout
func addAPIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("timeout", "10m", "How long to wait for responses")
	cmd.PersistentFlags().String("app", "https://app.overmind.tech", "The overmind instance to connect to.")
}

// Adds terraform-related flags to a command
func addTerraformBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("reset-stored-config", false, "[deprecated: this is now autoconfigured from local terraform files] Set this to reset the sources config stored in Overmind and input fresh values.")
	cmd.PersistentFlags().String("aws-config", "", "[deprecated: this is now autoconfigured from local terraform files] The chosen AWS config method, best set through the initial wizard when running the CLI. Options: 'profile_input', 'aws_profile', 'defaults', 'managed'.")
	cmd.PersistentFlags().String("aws-profile", "", "[deprecated: this is now autoconfigured from local terraform files] Set this to the name of the AWS profile to use.")
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("reset-stored-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-profile"))
	cmd.PersistentFlags().Bool("only-use-managed-sources", false, "Set this to skip local autoconfiguration and only use the managed sources as configured in Overmind.")
}
