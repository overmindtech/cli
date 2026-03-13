package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/muesli/reflow/wordwrap"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/knowledge"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// terraformPlanCmd represents the `terraform plan` command
var terraformPlanCmd = &cobra.Command{
	Use:    "plan [overmind options...] -- [terraform options...]",
	Short:  "Runs `terraform plan` and sends the results to Overmind to calculate a blast radius and risks.",
	PreRun: PreRunSetup,
	RunE:   TerraformPlan,
}

func TerraformPlan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	PTermSetup()

	hasPlanOutSet := false
	planFile := "overmind.plan"
	for i, a := range args {
		if a == "-out" || a == "--out=true" {
			hasPlanOutSet = true
			planFile = args[i+1]
		}
		if strings.HasPrefix(a, "-out=") {
			hasPlanOutSet = true
			planFile, _ = strings.CutPrefix(a, "-out=")
		}
		if strings.HasPrefix(a, "--out=") {
			hasPlanOutSet = true
			planFile, _ = strings.CutPrefix(a, "--out=")
		}
	}

	args = append([]string{"plan"}, args...)
	if !hasPlanOutSet {
		// if the user has not set a plan, we need to set a temporary file to
		// capture the output for the blast radius and risks calculation

		f, err := os.CreateTemp("", "overmind-plan")
		if err != nil {
			log.WithError(err).Fatal("failed to create temporary plan file")
		}

		planFile = f.Name()
		args = append(args, "-out", planFile)
		// TODO: remember whether we used a temporary plan file and remove it when done
	}

	ctx, oi, _, cleanup, err := StartSources(ctx, cmd, args)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return TerraformPlanImpl(ctx, cmd, oi, args, planFile)
}

func TerraformPlanImpl(ctx context.Context, cmd *cobra.Command, oi sdp.OvermindInstance, args []string, planFile string) error {
	span := trace.SpanFromContext(ctx)

	// this printer will be configured once the terraform plan command has
	// completed  and the terminal is available again
	postPlanPrinter := atomic.Pointer[pterm.MultiPrinter]{}

	revlinkPool := RunRevlinkWarmup(ctx, oi, &postPlanPrinter, args)

	err := RunPlan(ctx, args)
	if err != nil {
		return err
	}

	log.Debug("done running terraform plan")

	// start showing revlink warmup status now that the terminal is free
	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()

	// create a spinner for removing secrets before publishing `multi` to the
	// postPlanPrinter, so that "removing secrets" is shown before the revlink
	// status updates
	removingSecretsSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Removing secrets")
	postPlanPrinter.Store(&multi)

	///////////////////////////////////////////////////////////////////
	// Convert provided plan into JSON for easier parsing
	///////////////////////////////////////////////////////////////////

	tfPlanJsonCmd := exec.CommandContext(ctx, "terraform", "show", "-json", planFile)

	tfPlanJsonCmd.Stderr = multi.NewWriter() // send output through PTerm; is usually empty

	log.WithField("args", tfPlanJsonCmd.Args).Debug("converting plan to JSON")
	planJson, err := tfPlanJsonCmd.Output()
	if err != nil {
		removingSecretsSpinner.Fail(fmt.Sprintf("Removing secrets: %v", err))
		return fmt.Errorf("failed to convert terraform plan to JSON: %w", err)
	}

	removingSecretsSpinner.Success()

	// Detect the repository URL if it wasn't provided
	repoUrl := viper.GetString("repo")
	if repoUrl == "" {
		repoUrl, _ = DetectRepoURL(AllDetectors)
	}

	///////////////////////////////////////////////////////////////////
	// Extract changes from the plan and created mapped item diffs
	///////////////////////////////////////////////////////////////////

	resourceExtractionSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Extracting resources")
	resourceExtractionResults := multi.NewWriter()
	time.Sleep(200 * time.Millisecond) // give the UI a little time to update

	scope := tfutils.RepoToScope(repoUrl)

	// Map the terraform changes to Overmind queries
	mappingResponse, err := tfutils.MappedItemDiffsFromPlan(ctx, planJson, planFile, scope, log.Fields{})
	if err != nil {
		resourceExtractionSpinner.Fail(fmt.Sprintf("Removing secrets: %v", err))
		return nil
	}

	removingSecretsSpinner.Success(fmt.Sprintf("Removed %v secrets", mappingResponse.RemovedSecrets))

	resourceExtractionSpinner.UpdateText(fmt.Sprintf("Extracted %v changing resources: %v supported %v skipped %v unsupported %v pending creation\n",
		mappingResponse.NumTotal(),
		mappingResponse.NumSuccess(),
		mappingResponse.NumNotEnoughInfo(),
		mappingResponse.NumUnsupported(),
		mappingResponse.NumPendingCreation(),
	))

	// Sort the supported and unsupported changes so that they display nicely
	slices.SortFunc(mappingResponse.Results, func(a, b tfutils.PlannedChangeMapResult) int {
		return int(a.Status) - int(b.Status)
	})

	// render the list of supported and unsupported changes for the UI
	for _, mapping := range mappingResponse.Results {
		var printer pterm.PrefixPrinter
		switch mapping.Status {
		case tfutils.MapStatusSuccess:
			printer = pterm.Success
		case tfutils.MapStatusNotEnoughInfo:
			printer = pterm.Warning
		case tfutils.MapStatusUnsupported:
			printer = pterm.Error
		case tfutils.MapStatusPendingCreation:
			printer = pterm.Info
		}

		line := printer.Sprintf("%v (%v)", mapping.TerraformName, mapping.Message)
		_, err = fmt.Fprintf(resourceExtractionResults, "   %v\n", line)
		if err != nil {
			return fmt.Errorf("error writing to resource extraction results: %w", err)
		}
	}

	time.Sleep(200 * time.Millisecond) // give the UI a little time to update

	resourceExtractionSpinner.Success()

	// wait for the revlink warmup for 15 seconds. if it takes longer, we'll just continue
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- revlinkPool.Wait()
	}()

	select {
	case err = <-waitCh:
		if err != nil {
			return fmt.Errorf("error waiting for revlink warmup: %w", err)
		}
	case <-time.After(15 * time.Second):
		pterm.Info.Print("Done waiting for revlink warmup")
	}

	///////////////////////////////////////////////////////////////////
	// try to link up the plan with a Change and start submitting to the API
	///////////////////////////////////////////////////////////////////

	uploadChangesSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Uploading planned changes")

	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		ticketLink, err = getTicketLinkFromPlan(planFile)
		if err != nil {
			uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to get ticket link from plan: %v", err))
			return nil
		}
	}

	client := AuthenticatedChangesClient(ctx, oi)
	changeUuid, err := getChangeUUIDAndCheckStatus(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, false)
	if err != nil {
		uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed searching for existing changes: %v", err))
		return nil
	}

	title := changeTitle(ctx, viper.GetString("title"))
	tfPlanTextCmd := exec.CommandContext(ctx, "terraform", "show", planFile)

	tfPlanTextCmd.Stderr = multi.NewWriter() // send output through PTerm; is usually empty

	log.WithField("args", tfPlanTextCmd.Args).Debug("pretty-printing plan")
	tfPlanOutput, err := tfPlanTextCmd.Output()
	if err != nil {
		uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to pretty-print plan: %v", err))
		return nil
	}

	codeChangesOutput := tryLoadText(ctx, viper.GetString("code-changes-diff"))

	enrichedTags, err := parseTagsArgument()
	if err != nil {
		uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to parse tags: %v", err))
		return nil
	}

	labels, err := parseLabelsArgument()
	if err != nil {
		uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to parse labels: %v", err))
		return nil
	}

	properties := &sdp.ChangeProperties{
		Title:        title,
		Description:  viper.GetString("description"),
		TicketLink:   ticketLink,
		Owner:        viper.GetString("owner"),
		RawPlan:      string(tfPlanOutput),
		CodeChanges:  codeChangesOutput,
		Repo:         repoUrl,
		EnrichedTags: enrichedTags,
		Labels:       labels,
	}

	if changeUuid == uuid.Nil {
		uploadChangesSpinner.UpdateText("Uploading planned changes (new)")
		log.Debug("Creating a new change")
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: properties,
			},
		})
		if err != nil {
			uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to create a new change: %v", err))
			return nil
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			uploadChangesSpinner.Fail("Uploading planned changes: failed to read change id")
			return nil
		}

		changeUuid = *maybeChangeUuid
		span.SetAttributes(
			attribute.String("ovm.change.uuid", changeUuid.String()),
			attribute.Bool("ovm.change.new", true),
		)
	} else {
		uploadChangesSpinner.UpdateText("Uploading planned changes (update)")
		log.WithField("change", changeUuid).Debug("Updating an existing change")

		_, err := client.UpdateChange(ctx, &connect.Request[sdp.UpdateChangeRequest]{
			Msg: &sdp.UpdateChangeRequest{
				UUID:       changeUuid[:],
				Properties: properties,
			},
		})
		if err != nil {
			uploadChangesSpinner.Fail(fmt.Sprintf("Uploading planned changes: failed to update change: %v", err))
			return nil
		}
	}
	time.Sleep(200 * time.Millisecond) // give the UI a little time to update
	uploadChangesSpinner.Success()

	///////////////////////////////////////////////////////////////////
	// Upload the planned changes to the API
	///////////////////////////////////////////////////////////////////

	uploadPlannedChange, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Uploading planned changes")
	log.WithField("change", changeUuid).Debug("Uploading planned changes")

	// Discover and convert knowledge files
	knowledgeDir := knowledge.FindKnowledgeDir(".")
	sdpKnowledge := knowledge.DiscoverAndConvert(ctx, knowledgeDir)

	_, err = client.StartChangeAnalysis(ctx, &connect.Request[sdp.StartChangeAnalysisRequest]{
		Msg: &sdp.StartChangeAnalysisRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: mappingResponse.GetItemDiffs(),
			Knowledge:     sdpKnowledge,
		},
	})
	if err != nil {
		uploadPlannedChange.Fail(fmt.Sprintf("Uploading planned changes: failed to update: %v", err))
		return nil
	}
	uploadPlannedChange.Success("Uploaded planned changes: Done")

	changeUrl := *oi.FrontendUrl
	changeUrl.Path = fmt.Sprintf("%v/changes/%v", changeUrl.Path, changeUuid)
	log.WithField("change-url", changeUrl.String()).Info("Change ready")

	///////////////////////////////////////////////////////////////////
	// wait for change analysis to complete (poll GetChange by change_analysis_status)
	///////////////////////////////////////////////////////////////////
	changeAnalysisSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Change Analysis")

retryLoop:
	for {
		changeRes, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
			Msg: &sdp.GetChangeRequest{
				UUID: changeUuid[:],
			},
		})
		if err != nil {
			changeAnalysisSpinner.Fail(fmt.Sprintf("Change Analysis failed to get change: %v", err))
			return fmt.Errorf("failed to get change during change analysis: %w", err)
		}
		if changeRes.Msg == nil || changeRes.Msg.GetChange() == nil {
			changeAnalysisSpinner.Fail("Change Analysis failed: received empty change response")
			return fmt.Errorf("change analysis failed: received empty change response")
		}
		ch := changeRes.Msg.GetChange()
		md := ch.GetMetadata()
		if md == nil || md.GetChangeAnalysisStatus() == nil {
			changeAnalysisSpinner.Fail("Change Analysis failed: change metadata or analysis status missing")
			return fmt.Errorf("change analysis failed: change metadata or change analysis status is nil")
		}
		status := md.GetChangeAnalysisStatus().GetStatus()
		switch status {
		case sdp.ChangeAnalysisStatus_STATUS_DONE, sdp.ChangeAnalysisStatus_STATUS_SKIPPED:
			changeAnalysisSpinner.Success()
			break retryLoop
		case sdp.ChangeAnalysisStatus_STATUS_ERROR:
			changeAnalysisSpinner.Fail("Change analysis failed")
			return fmt.Errorf("change analysis completed with error status")
		case sdp.ChangeAnalysisStatus_STATUS_UNSPECIFIED, sdp.ChangeAnalysisStatus_STATUS_INPROGRESS:
			// keep polling
		}
		time.Sleep(3 * time.Second)
		if ctx.Err() != nil {
			changeAnalysisSpinner.Fail("Cancelled")
			return ctx.Err()
		}
	}
	risksRes, err := client.GetChangeRisks(ctx, &connect.Request[sdp.GetChangeRisksRequest]{
		Msg: &sdp.GetChangeRisksRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get calculated risks: %w", err)
	}
	if risksRes.Msg == nil {
		return fmt.Errorf("failed to get calculated risks: response message was nil")
	}
	if risksRes.Msg.GetChangeRiskMetadata() == nil {
		return fmt.Errorf("failed to get calculated risks: change risk metadata was nil")
	}
	calculatedRisks := risksRes.Msg.GetChangeRiskMetadata().GetRisks()
	// Submit milestone for tracing
	if cmdSpan != nil {
		cmdSpan.AddEvent("Change Analysis finished", trace.WithAttributes(
			attribute.Int("ovm.risks.count", len(calculatedRisks)),
			attribute.String("ovm.change.uuid", changeUuid.String()),
		))
	}

	bits := []string{}
	bits = append(bits, "")
	bits = append(bits, "")
	if len(calculatedRisks) == 0 {
		bits = append(bits, styleH1().Render("Potential Risks"))
		bits = append(bits, "")
		bits = append(bits, "Overmind has not identified any risks associated with this change.")
		bits = append(bits, "")
		bits = append(bits, "This could be due to the change being low risk with no impact on other parts of the system, or involving resources that Overmind currently does not support.")
	} else if changeUrl.String() != "" {
		bits = append(bits, styleH1().Render("Potential Risks"))
		bits = append(bits, "")
		for _, r := range calculatedRisks {
			severity := ""
			switch r.GetSeverity() {
			case sdp.Risk_SEVERITY_HIGH:
				severity = lipgloss.NewStyle().
					Background(ColorPalette.BgDanger).
					Foreground(ColorPalette.LabelTitle).
					Padding(0, 1).
					Bold(true).
					Render("High ‼")
			case sdp.Risk_SEVERITY_MEDIUM:
				severity = lipgloss.NewStyle().
					Background(ColorPalette.BgWarning).
					Foreground(ColorPalette.LabelTitle).
					Padding(0, 1).
					Render("Medium !")
			case sdp.Risk_SEVERITY_LOW:
				severity = lipgloss.NewStyle().
					Background(ColorPalette.LabelBase).
					Foreground(ColorPalette.LabelTitle).
					Padding(0, 1).
					Render("Low ⓘ ")
			case sdp.Risk_SEVERITY_UNSPECIFIED:
				// do nothing
			}
			title := lipgloss.NewStyle().
				Foreground(ColorPalette.BgMain).
				PaddingRight(1).
				Bold(true).
				Render(r.GetTitle())

			bits = append(bits, fmt.Sprintf("%v%v\n\n%v",
				title,
				severity,
				wordwrap.String(r.GetDescription(), min(160, pterm.GetTerminalWidth()-4))))

			riskUUID, _ := uuid.FromBytes(r.GetUUID())
			riskURL := fmt.Sprintf("%v/blast-radius?selectedRisk=%v&utm_source=cli&cli_version=%v", changeUrl.String(), riskUUID.String(), tracing.Version())
			bits = append(bits, fmt.Sprintf("%v\n\n", osc8Hyperlink(riskURL, "View risk ↗")))
		}
		changeURLWithUTM := fmt.Sprintf("%v?utm_source=cli&cli_version=%v", changeUrl.String(), tracing.Version())
		bits = append(bits, fmt.Sprintf("\nView the blast radius graph and risks:\n%v\n\n", osc8Hyperlink(changeURLWithUTM, "Open in Overmind ↗")))
	}

	pterm.Fprintln(multi.NewWriter(), strings.Join(bits, "\n"))

	return nil
}

// supportsOSCHyperlinks checks if the terminal likely supports OSC 8 hyperlinks.
// Combines a TTY check with environment-based heuristics.
func supportsOSCHyperlinks() bool {
	if fi, err := os.Stdout.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return envSupportsOSCHyperlinks()
}

// envSupportsOSCHyperlinks checks environment variables to determine if the terminal
// likely supports OSC 8 hyperlinks. Split out from supportsOSCHyperlinks so that tests
// can exercise the env heuristics in isolation — go test pipes stdout, so the
// TTY check in supportsOSCHyperlinks always fails under test.
func envSupportsOSCHyperlinks() bool {
	if os.Getenv("CI") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "dumb" {
		return false
	}
	if strings.HasPrefix(os.Getenv("TERM"), "screen") && os.Getenv("TMUX") == "" {
		return false
	}
	if os.Getenv("TERM_PROGRAM") != "" {
		return true
	}
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}
	if os.Getenv("TERM") == "xterm-kitty" {
		return true
	}
	if strings.Contains(os.Getenv("TERM"), "256color") {
		return true
	}
	return false
}

// osc8Hyperlink returns an OSC 8 hyperlink if the terminal supports it, otherwise
// the raw URL. Supported by iTerm2, GNOME Terminal, Windows Terminal, WezTerm,
// kitty, Alacritty; degrades gracefully in unsupported terminals.
func osc8Hyperlink(url, text string) string {
	if supportsOSCHyperlinks() {
		return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
	}
	return url
}

// getTicketLinkFromPlan reads the plan file to create a unique hash to identify this change
func getTicketLinkFromPlan(planFile string) (string, error) {
	plan, err := os.ReadFile(planFile)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file (%v): %w", planFile, err)
	}
	h := sha256.New()
	h.Write(plan)
	return fmt.Sprintf("tfplan://{SHA256}%x", h.Sum(nil)), nil
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
	addChangeUuidFlags(terraformPlanCmd)
	addTerraformBaseFlags(terraformPlanCmd)
}
