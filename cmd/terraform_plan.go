package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/muesli/reflow/wordwrap"
	"github.com/overmindtech/cli/cmd/datamaps"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	Run: CmdWrapper("plan", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfPlanModel),
}

type tfPlanModel struct {
	ctx context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance

	args       []string
	planTask   taskModel
	planHeader string

	revlinkWarmupFinished bool

	runTfPlan        bool
	tfPlanFinished   bool
	processing       chan tea.Msg
	blastRadiusModel snapshotModel
	progress         []string
	changeUrl        string

	riskTask           taskModel
	blastRadiusItems   uint32
	blastRadiusEdges   uint32
	riskMilestones     []*sdp.RiskCalculationStatus_ProgressMilestone
	riskMilestoneTasks []taskModel
	risks              []*sdp.Risk

	fatalError string
	width      int
}

// assert interface
var _ FinalReportingModel = (*tfPlanModel)(nil)

type triggerTfPlanMsg struct{}
type tfPlanFinishedMsg struct{}
type triggerPlanProcessingMsg struct{}
type processingActivityMsg struct{ text string }
type changeUpdatedMsg struct {
	url            string
	riskMilestones []*sdp.RiskCalculationStatus_ProgressMilestone
	risks          []*sdp.Risk
}
type processingFinishedActivityMsg struct{ text string }
type delayQuitMsg struct{}

func NewTfPlanModel(args []string) tea.Model {
	args = append([]string{"plan"}, args...)
	// -out needs to go last to override whatever the user specified on the command line
	args = append(args, "-out", "overmind.plan")

	planHeader := `Running ` + "`" + `terraform %v` + "`\n"
	planHeader = fmt.Sprintf(planHeader, strings.Join(args, " "))

	return tfPlanModel{
		args:       args,
		planTask:   NewTaskModel("Planning Changes"),
		planHeader: planHeader,

		processing:       make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		blastRadiusModel: NewSnapShotModel("Calculating Blast Radius"),
		progress:         []string{},

		riskTask: NewTaskModel("Calculating Risks"),
	}
}

func (m tfPlanModel) Init() tea.Cmd {
	return tea.Batch(
		m.planTask.Init(),
		m.blastRadiusModel.Init(),
		m.riskTask.Init(),
	)
}

func (m tfPlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debugf("tfPlanModel: Update %T received %+v", msg, msg)

	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case sourcesInitialisedMsg:
		m.runTfPlan = true
		m.planTask.status = taskStatusRunning
		// defer the actual command to give the view a chance to show the header
		cmds = append(cmds, func() tea.Msg { return triggerTfPlanMsg{} })

	case triggerTfPlanMsg:
		c := exec.CommandContext(m.ctx, "terraform", m.args...) // nolint:gosec // this is a user-provided command, let them do their thing

		// inject the profile, if configured
		if aws_profile := viper.GetString("aws-profile"); aws_profile != "" {
			c.Env = append(c.Env, fmt.Sprintf("AWS_PROFILE=%v", aws_profile))
		}

		m.blastRadiusModel.state = "executing terraform plan"

		if viper.GetString("ovm-test-fake") != "" {
			c = exec.CommandContext(m.ctx, "bash", "-c", "for i in $(seq 100); do echo fake terraform plan progress line $i of 100; done; sleep 1")
		}

		cmds = append(cmds, tea.ExecProcess(
			c,
			func(err error) tea.Msg {
				if err != nil {
					return fatalError{err: fmt.Errorf("failed to run terraform plan: %w", err)}
				}

				return tfPlanFinishedMsg{}
			}))

	case revlinkWarmupFinishedMsg:
		m.revlinkWarmupFinished = true
		if m.tfPlanFinished {
			cmds = append(cmds, func() tea.Msg { return triggerPlanProcessingMsg{} })
		}
	case tfPlanFinishedMsg:
		m.tfPlanFinished = true
		m.planTask.status = taskStatusDone

		if m.revlinkWarmupFinished {
			cmds = append(cmds, func() tea.Msg { return triggerPlanProcessingMsg{} })
		}

	case triggerPlanProcessingMsg:
		m.blastRadiusModel.status = taskStatusRunning
		m.blastRadiusModel.state = "executed terraform plan"

		cmds = append(cmds,
			m.processPlanCmd,
			m.blastRadiusModel.spinner.Tick,
			m.waitForProcessingActivity,
		)
	case processingActivityMsg:
		m.blastRadiusModel.state = "processing"
		m.progress = append(m.progress, msg.text)
		cmds = append(cmds, m.waitForProcessingActivity)
	case processingFinishedActivityMsg:
		m.blastRadiusModel.status = taskStatusDone
		m.blastRadiusModel.state = "finished"
		m.riskTask.status = taskStatusDone
		m.progress = append(m.progress, msg.text)
		cmds = append(cmds, m.waitForProcessingActivity)
	case changeUpdatedMsg:
		m.changeUrl = msg.url
		m.riskMilestones = msg.riskMilestones
		if len(m.riskMilestoneTasks) != len(msg.riskMilestones) {
			m.riskMilestoneTasks = []taskModel{}
			for _, ms := range msg.riskMilestones {
				tm := NewTaskModel(ms.GetDescription())
				m.riskMilestoneTasks = append(m.riskMilestoneTasks, tm)
				cmds = append(cmds, tm.Init())
			}
		}
		for i, ms := range msg.riskMilestones {
			m.riskMilestoneTasks[i].title = ms.GetDescription()
			switch ms.GetStatus() {
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_PENDING:
				m.riskMilestoneTasks[i].status = taskStatusPending
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_ERROR:
				m.riskMilestoneTasks[i].status = taskStatusError
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE:
				m.riskMilestoneTasks[i].status = taskStatusDone
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_INPROGRESS:
				m.riskMilestoneTasks[i].status = taskStatusRunning
				cmds = append(cmds, m.riskMilestoneTasks[i].spinner.Tick)
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_SKIPPED:
				m.riskMilestoneTasks[i].status = taskStatusSkipped
			}
		}
		m.risks = msg.risks

		if len(m.riskMilestones) > 0 {
			m.riskTask.status = taskStatusRunning
			cmds = append(cmds, m.riskTask.spinner.Tick)
		} else if len(m.risks) > 0 {
			m.riskTask.status = taskStatusDone
		} else {
			var allSkipped = true
			for _, ms := range m.riskMilestoneTasks {
				if ms.status != taskStatusSkipped {
					allSkipped = false
					break
				}
			}
			if allSkipped {
				m.riskTask.status = taskStatusSkipped
			}
		}

		m.blastRadiusModel.status = taskStatusDone
		m.blastRadiusModel.state = "Change updated"
		cmds = append(cmds, m.waitForProcessingActivity)

	case startSnapshotMsg:
		var cmd tea.Cmd
		m.blastRadiusModel, cmd = m.blastRadiusModel.Update(msg)
		cmds = append(cmds, m.waitForProcessingActivity, cmd)
	case progressSnapshotMsg:
		m.blastRadiusItems = msg.items
		m.blastRadiusEdges = msg.edges

		var cmd tea.Cmd
		m.blastRadiusModel, cmd = m.blastRadiusModel.Update(msg)
		cmds = append(cmds, m.waitForProcessingActivity, cmd)
	case finishSnapshotMsg:
		var cmd tea.Cmd
		m.blastRadiusModel, cmd = m.blastRadiusModel.Update(msg)
		cmds = append(cmds, tea.Sequence(cmd, func() tea.Msg { return delayQuitMsg{} }))
	case delayQuitMsg:
		cmds = append(cmds, tea.Quit)

	case fatalError:
		m.fatalError = msg.err.Error()
		cmds = append(cmds, tea.Quit)
	default:
		var cmd tea.Cmd
		m.planTask, cmd = m.planTask.Update(msg)
		cmds = append(cmds, cmd)

		m.blastRadiusModel, cmd = m.blastRadiusModel.Update(msg)
		cmds = append(cmds, cmd)

		m.riskTask, cmd = m.riskTask.Update(msg)
		cmds = append(cmds, cmd)

		for i, ms := range m.riskMilestoneTasks {
			m.riskMilestoneTasks[i], cmd = ms.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m tfPlanModel) View() string {
	bits := []string{}

	if m.planTask.status != taskStatusPending {
		bits = append(bits, m.planTask.View())
	}

	if m.runTfPlan && !m.tfPlanFinished {
		bits = append(bits, markdownToString(m.planHeader))
	}

	if m.blastRadiusModel.status != taskStatusPending {
		bits = append(bits, m.blastRadiusModel.View())
	}

	if m.riskTask.status != taskStatusPending {
		bits = append(bits, m.riskTask.View())
	}

	if m.changeUrl != "" {
		for _, t := range m.riskMilestoneTasks {
			bits = append(bits, fmt.Sprintf("   %v", t.View()))
		}
		bits = append(bits, fmt.Sprintf("\nCheck the blast radius graph at:\n%v\n\n", m.changeUrl))
	}

	return strings.Join(bits, "\n") + "\n"
}

func (m tfPlanModel) FinalReport() string {
	bits := []string{}
	if m.blastRadiusItems > 0 {
		bits = append(bits, "")
		bits = append(bits, styleH1().Render("Blast Radius"))
		bits = append(bits, fmt.Sprintf("\nItems: %v\nEdges: %v\n", m.blastRadiusItems, m.blastRadiusEdges))
	}
	if m.changeUrl != "" && len(m.risks) > 0 {
		bits = append(bits, "")
		bits = append(bits, styleH1().Render("Potential Risks"))
		bits = append(bits, "")
		for _, r := range m.risks {
			severity := ""
			switch r.GetSeverity() {
			case sdp.Risk_SEVERITY_HIGH:
				severity = lipgloss.NewStyle().Background(ColorPalette.BgDanger).Render("  High 🔥  ")
			case sdp.Risk_SEVERITY_MEDIUM:
				severity = lipgloss.NewStyle().Background(ColorPalette.BgWarning).Render("  Medium ❗  ")
			case sdp.Risk_SEVERITY_LOW:
				severity = lipgloss.NewStyle().Background(ColorPalette.LabelTitle).Render("  Low ℹ️  ")
			case sdp.Risk_SEVERITY_UNSPECIFIED:
				// do nothing
			}
			bits = append(bits, (fmt.Sprintf("%v %v\n\n%v\n\n",
				severity,
				styleH2().Render(r.GetTitle()),
				wordwrap.String(r.GetDescription(), min(160, m.width-4)))))
		}
		bits = append(bits, fmt.Sprintf("\nCheck the blast radius graph and risks at:\n%v\n\n", m.changeUrl))
	}
	return strings.Join(bits, "\n") + "\n"
}

// A command that waits for the activity on the processing channel.
func (m tfPlanModel) waitForProcessingActivity() tea.Msg {
	msg := <-m.processing
	log.Debugf("waitForProcessingActivity received %T: %+v", msg, msg)
	return msg
}

func (m tfPlanModel) processPlanCmd() tea.Msg {
	ctx := m.ctx
	span := trace.SpanFromContext(ctx)

	m.processing <- startSnapshotMsg{newState: "converting terraform plan to JSON"}

	if viper.GetString("ovm-test-fake") != "" {
		m.processing <- processingActivityMsg{"Fake processing json plan"}
		time.Sleep(time.Second)
		m.processing <- processingActivityMsg{"Fake creating a new change"}
		time.Sleep(time.Second)
		m.processing <- progressSnapshotMsg{newState: "fake processing"}
		time.Sleep(time.Second)
		m.processing <- changeUpdatedMsg{url: "https://example.com/changes/abc"}
		time.Sleep(time.Second)

		m.processing <- processingActivityMsg{"Fake CalculateBlastRadiusResponse Status update: progress"}
		time.Sleep(time.Second)

		m.processing <- progressSnapshotMsg{
			newState: "discovering blast radius",
			items:    10,
			edges:    21,
		}
		time.Sleep(time.Second)

		m.processing <- changeUpdatedMsg{url: "https://example.com/changes/abc"}
		m.processing <- processingActivityMsg{"Calculating risks"}
		time.Sleep(time.Second)

		m.processing <- changeUpdatedMsg{
			url: "https://example.com/changes/abc",
			riskMilestones: []*sdp.RiskCalculationStatus_ProgressMilestone{
				{
					Description: "fake done milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_INPROGRESS,
				},
				{
					Description: "fake inprogress milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_PENDING,
				},
				{
					Description: "fake pending milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_PENDING,
				},
			},
			risks: []*sdp.Risk{},
		}
		time.Sleep(1500 * time.Millisecond)

		m.processing <- changeUpdatedMsg{
			url: "https://example.com/changes/abc",
			riskMilestones: []*sdp.RiskCalculationStatus_ProgressMilestone{
				{
					Description: "fake done milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
				{
					Description: "fake inprogress milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_INPROGRESS,
				},
				{
					Description: "fake pending milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_PENDING,
				},
			},
			risks: []*sdp.Risk{},
		}
		time.Sleep(1500 * time.Millisecond)

		m.processing <- changeUpdatedMsg{
			url: "https://example.com/changes/abc",
			riskMilestones: []*sdp.RiskCalculationStatus_ProgressMilestone{
				{
					Description: "fake done milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
				{
					Description: "fake inprogress milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
				{
					Description: "fake pending milestone",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_INPROGRESS,
				},
			},
			risks: []*sdp.Risk{},
		}
		time.Sleep(1500 * time.Millisecond)

		high := uuid.New()
		medium := uuid.New()
		low := uuid.New()
		m.processing <- changeUpdatedMsg{
			url: "https://example.com/changes/abc",
			riskMilestones: []*sdp.RiskCalculationStatus_ProgressMilestone{
				{
					Description: "fake done milestone - done",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
				{
					Description: "fake inprogress milestone - done",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
				{
					Description: "fake pending milestone - done",
					Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_DONE,
				},
			},
			risks: []*sdp.Risk{
				{
					UUID:         high[:],
					Title:        "fake high risk titled risk",
					Severity:     sdp.Risk_SEVERITY_HIGH,
					Description:  TEST_RISK,
					RelatedItems: []*sdp.Reference{},
				},
				{
					UUID:         medium[:],
					Title:        "fake medium risk titled risk",
					Severity:     sdp.Risk_SEVERITY_MEDIUM,
					Description:  TEST_RISK,
					RelatedItems: []*sdp.Reference{},
				},
				{
					UUID:         low[:],
					Title:        "fake low risk titled risk",
					Severity:     sdp.Risk_SEVERITY_LOW,
					Description:  TEST_RISK,
					RelatedItems: []*sdp.Reference{},
				},
			},
		}
		time.Sleep(time.Second)

		m.processing <- processingFinishedActivityMsg{"Fake done"}
		time.Sleep(time.Second)
		return finishSnapshotMsg{newState: "fake done"}
	}

	tfPlanJsonCmd := exec.CommandContext(ctx, "terraform", "show", "-json", "overmind.plan")
	tfPlanJsonCmd.Stderr = os.Stderr // TODO: capture and output this through the View() instead

	planJson, err := tfPlanJsonCmd.Output()
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed to convert terraform plan to JSON: %w", err)}
	}

	plannedChanges, err := mappedItemDiffsFromPlan(ctx, planJson, "overmind.plan", log.Fields{})
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed to parse terraform plan: %w", err)}
	}

	m.processing <- processingActivityMsg{"converted terraform plan to JSON"}
	m.processing <- progressSnapshotMsg{newState: "converted terraform plan to JSON"}

	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		ticketLink, err = getTicketLinkFromPlan()
		if err != nil {
			close(m.processing)
			return err
		}
	}

	client := AuthenticatedChangesClient(ctx, m.oi)
	changeUuid, err := getChangeUuid(ctx, m.oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, false)
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed searching for existing changes: %w", err)}
	}

	title := changeTitle(viper.GetString("title"))
	tfPlanOutput := tryLoadText(ctx, viper.GetString("terraform-plan-output"))
	codeChangesOutput := tryLoadText(ctx, viper.GetString("code-changes-diff"))

	if changeUuid == uuid.Nil {
		m.processing <- processingActivityMsg{"Creating a new change"}
		m.processing <- progressSnapshotMsg{newState: "creating a new change"}
		log.Debug("Creating a new change")
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
			close(m.processing)
			return fatalError{err: fmt.Errorf("processPlanCmd: failed to create a new change: %w", err)}
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			close(m.processing)
			return fatalError{err: fmt.Errorf("processPlanCmd: failed to read change id: %w", err)}
		}

		changeUuid = *maybeChangeUuid
		span.SetAttributes(
			attribute.String("ovm.change.uuid", changeUuid.String()),
			attribute.Bool("ovm.change.new", true),
		)
	} else {
		m.processing <- processingActivityMsg{"Updating an existing change"}
		m.processing <- progressSnapshotMsg{newState: "updating an existing change"}
		log.WithField("change", changeUuid).Debug("Updating an existing change")
		span.SetAttributes(
			attribute.String("ovm.change.uuid", changeUuid.String()),
			attribute.Bool("ovm.change.new", false),
		)

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
			close(m.processing)
			return fatalError{err: fmt.Errorf("processPlanCmd: failed to update change: %w", err)}
		}
	}

	m.processing <- processingActivityMsg{"Uploading planned changes"}
	log.WithField("change", changeUuid).Debug("Uploading planned changes")
	m.processing <- progressSnapshotMsg{newState: "uploading planned changes"}

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: plannedChanges,
		},
	})
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed to update planned changes: %w", err)}
	}

	last_log := time.Now()
	first_log := true
	var msg *sdp.CalculateBlastRadiusResponse
	for resultStream.Receive() {
		msg = resultStream.Msg()

		// log the first message and at most every 250ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.GetState() != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithField("msg", msg).Trace("Status update")
			last_log = time.Now()
			first_log = false
		}
		m.processing <- processingActivityMsg{fmt.Sprintf("Status update: %v", msg)}
		stateLabel := "unknown"
		switch msg.GetState() {
		case sdp.CalculateBlastRadiusResponse_STATE_UNSPECIFIED:
			stateLabel = "unknown"
		case sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING:
			stateLabel = "discovering blast radius"
		case sdp.CalculateBlastRadiusResponse_STATE_FINDING_APPS:
			stateLabel = "finding apps"
		case sdp.CalculateBlastRadiusResponse_STATE_SAVING:
			stateLabel = "saving blast radius"
		case sdp.CalculateBlastRadiusResponse_STATE_DONE:
			stateLabel = "done"
		}
		m.processing <- progressSnapshotMsg{
			newState: stateLabel,
			items:    msg.GetNumItems(),
			edges:    msg.GetNumEdges(),
		}
	}
	if resultStream.Err() != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: error streaming results: %w", err)}
	}

	changeUrl := *m.oi.FrontendUrl
	changeUrl.Path = fmt.Sprintf("%v/changes/%v/blast-radius", changeUrl.Path, changeUuid)
	log.WithField("change-url", changeUrl.String()).Info("Change ready")

	m.processing <- changeUpdatedMsg{url: changeUrl.String()}

	// wait for risk calculation to happen
	m.processing <- processingActivityMsg{"Calculating risks"}
	for {
		riskRes, err := client.GetChangeRisks(ctx, &connect.Request[sdp.GetChangeRisksRequest]{
			Msg: &sdp.GetChangeRisksRequest{
				UUID: changeUuid[:],
			},
		})
		if err != nil {
			close(m.processing)
			return fatalError{err: fmt.Errorf("processPlanCmd: failed to get change risks: %w", err)}
		}

		m.processing <- changeUpdatedMsg{
			url:            changeUrl.String(),
			riskMilestones: riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetProgressMilestones(),
			risks:          riskRes.Msg.GetChangeRiskMetadata().GetRisks(),
		}

		if riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetStatus() == sdp.RiskCalculationStatus_STATUS_INPROGRESS {
			time.Sleep(time.Second)
			// retry
		} else {
			// it's done (or errored)
			break
		}

		if ctx.Err() != nil {
			return fatalError{err: fmt.Errorf("processPlanCmd: context cancelled: %w", ctx.Err())}
		}

	}

	m.processing <- processingFinishedActivityMsg{"Done"}
	return finishSnapshotMsg{
		newState: "calculated blast radius and risks",
		items:    msg.GetNumItems(),
		edges:    msg.GetNumEdges(),
	}
}

// getTicketLinkFromPlan reads the plan file to create a unique hash to identify this change
func getTicketLinkFromPlan() (string, error) {
	plan, err := os.ReadFile("overmind.plan")
	if err != nil {
		return "", fmt.Errorf("failed to read overmind.plan file: %w", err)
	}
	h := sha256.New()
	h.Write(plan)
	return fmt.Sprintf("tfplan://{SHA256}%x", h.Sum(nil)), nil
}

func mappedItemDiffsFromPlanFile(ctx context.Context, fileName string, lf log.Fields) ([]*sdp.MappedItemDiff, error) {
	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return nil, err
	}

	return mappedItemDiffsFromPlan(ctx, planJSON, fileName, lf)
}

// mappedItemDiffsFromPlan takes a plan JSON, file name, and log fields as input
// and returns a slice of mapped item differences and an error. It parses the
// plan JSON, extracts resource changes, and creates mapped item differences for
// each resource change. It also generates mapping queries based on the resource
// type and current resource values. The function categorizes the mapped item
// differences into supported and unsupported changes. Finally, it logs the
// number of supported and unsupported changes and returns the mapped item
// differences.
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

		// Load mappings for this type. These mappings tell us how to create an
		// SDP query that will return this resource
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
					WithField("terraform-query-field", mapData.QueryField).Warn("Adding unmapped resource")
				plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: nil, // unmapped item has no mapping query
				})
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

func addTerraformBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("reset-stored-config", false, "Set this to reset the sources config stored in Overmind and input fresh values.")
	cmd.PersistentFlags().String("aws-config", "", "The chosen AWS config method, best set through the initial wizard when running the CLI. Options: 'profile_input', 'aws_profile', 'defaults', 'managed'.")
	cmd.PersistentFlags().String("aws-profile", "", "Set this to the name of the AWS profile to use.")
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
	addChangeUuidFlags(terraformPlanCmd)
	addTerraformBaseFlags(terraformPlanCmd)
}

const TEST_RISK = `In publishing and graphic design, Lorem ipsum (/ˌlɔː.rəm ˈɪp.səm/) is a placeholder text commonly used to demonstrate the visual form of a document or a typeface without relying on meaningful content. Lorem ipsum may be used as a placeholder before the final copy is available. It is also used to temporarily replace text in a process called greeking, which allows designers to consider the form of a webpage or publication, without the meaning of the text influencing the design.

Lorem ipsum is typically a corrupted version of De finibus bonorum et malorum, a 1st-century BC text by the Roman statesman and philosopher Cicero, with words altered, added, and removed to make it nonsensical and improper Latin. The first two words themselves are a truncation of dolorem ipsum ("pain itself").`
