package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/knowledge"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ErrInvalidKnowledgeFiles is returned when one or more knowledge files are invalid/skipped.
// Used so "knowledge list" can exit non-zero in CI when invalid files are found.
var ErrInvalidKnowledgeFiles = errors.New("invalid knowledge files found")

// knowledgeListCmd represents the knowledge list command
var knowledgeListCmd = &cobra.Command{
	Use:    "list",
	Short:  "Lists knowledge files that would be used from the current location",
	PreRun: PreRunSetup,
	RunE:   KnowledgeList,
}

func KnowledgeList(cmd *cobra.Command, args []string) error {
	startDir := viper.GetString("dir")
	explicitDirs := viper.GetStringSlice("knowledge-dir")
	output, err := renderKnowledgeList(startDir, explicitDirs)
	fmt.Print(output)
	if err != nil {
		return err
	}
	return nil
}

// renderKnowledgeList handles the knowledge list logic and returns formatted output.
// This is separated from the command for testability.
// If explicitDirs is provided, uses those directories; otherwise falls back to auto-discovery.
func renderKnowledgeList(startDir string, explicitDirs []string) (string, error) {
	var output strings.Builder

	knowledgeDirs := knowledge.ResolveKnowledgeDirs(startDir, explicitDirs)

	if len(knowledgeDirs) == 0 {
		output.WriteString(pterm.Info.Sprint("No .overmind/knowledge/ directory found from current location\n\n"))
		output.WriteString("Knowledge files help Overmind understand your infrastructure context.\n")
		output.WriteString("Create a .overmind/knowledge/ directory to add knowledge files.\n")
		output.WriteString("Without knowledge files, 'terraform plan' will proceed with standard analysis.\n")
		return output.String(), nil
	}

	files, warnings := knowledge.Discover(knowledgeDirs...)

	// Show resolved directories
	if len(knowledgeDirs) == 1 {
		output.WriteString(pterm.Info.Sprintf("Knowledge directory: %s\n\n", knowledgeDirs[0]))
	} else {
		output.WriteString(pterm.Info.Sprint("Knowledge directories (later overrides earlier):\n"))
		for i, dir := range knowledgeDirs {
			output.WriteString(pterm.Info.Sprintf("  %d. %s\n", i+1, dir))
		}
		output.WriteString("\n")
	}

	// Show valid files
	if len(files) > 0 {
		output.WriteString(pterm.DefaultHeader.Sprint("Valid Knowledge Files") + "\n\n")

		// Create table data with Source Dir column when multiple directories
		var tableData pterm.TableData
		if len(knowledgeDirs) > 1 {
			tableData = pterm.TableData{
				{"Name", "Description", "File Path", "Source Dir"},
			}
		} else {
			tableData = pterm.TableData{
				{"Name", "Description", "File Path"},
			}
		}

		for _, f := range files {
			if len(knowledgeDirs) > 1 {
				tableData = append(tableData, []string{
					f.Name,
					truncateDescription(f.Description, 60),
					f.FileName,
					f.SourceDir,
				})
			} else {
				tableData = append(tableData, []string{
					f.Name,
					truncateDescription(f.Description, 60),
					f.FileName,
				})
			}
		}

		table, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
		if err != nil {
			return "", fmt.Errorf("failed to render table: %w", err)
		}
		output.WriteString(table)
		output.WriteString("\n")
	} else if len(warnings) == 0 {
		output.WriteString(pterm.Info.Sprint("No knowledge files found\n\n"))
	}

	// Show warnings
	if len(warnings) > 0 {
		output.WriteString(pterm.DefaultHeader.Sprint("Invalid/Skipped Files") + "\n\n")

		for _, w := range warnings {
			output.WriteString(pterm.Warning.Sprintf("  %s\n", w.Path))
			fmt.Fprintf(&output, "    Reason: %s\n", w.Reason)
		}
		output.WriteString("\n")
		return output.String(), fmt.Errorf("%w (%d file(s))", ErrInvalidKnowledgeFiles, len(warnings))
	}

	return output.String(), nil
}

// truncateDescription truncates a description to maxLen characters, adding "..." if truncated
func truncateDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	return desc[:maxLen-3] + "..."
}

func init() {
	knowledgeCmd.AddCommand(knowledgeListCmd)

	knowledgeListCmd.Flags().String("dir", ".", "Directory to start searching from")
	cobra.CheckErr(knowledgeListCmd.Flags().MarkHidden("dir"))
	knowledgeListCmd.Flags().StringSlice("knowledge-dir", []string{}, "Knowledge directory paths to load. Can be specified multiple times or comma-separated. If not specified, auto-discovers .overmind/knowledge/ by walking up from the current directory.")
}
