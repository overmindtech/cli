package cmd

import (
	"context"
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
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type submitPlanModel struct {
	ctx context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance

	planFile string

	processing chan tea.Msg
	progress   []string
	changeUrl  string

	blastRadiusTask  snapshotModel
	blastRadiusItems uint32
	blastRadiusEdges uint32

	riskTask           taskModel
	riskMilestones     []*sdp.RiskCalculationStatus_ProgressMilestone
	riskMilestoneTasks []taskModel
	risks              []*sdp.Risk

	width int
}
type submitPlanNowMsg struct{}

type submitPlanUpdateMsg struct{ text string }
type submitPlanFinishedMsg struct{ text string }

type changeUpdatedMsg struct {
	url            string
	riskMilestones []*sdp.RiskCalculationStatus_ProgressMilestone
	risks          []*sdp.Risk
}

func NewSubmitPlanModel(planFile string) submitPlanModel {
	return submitPlanModel{
		planFile: planFile,

		processing: make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		progress:   []string{},

		blastRadiusTask: NewSnapShotModel("Calculating Blast Radius"),
		riskTask:        NewTaskModel("Calculating Risks"),
	}
}

func (m submitPlanModel) Init() tea.Cmd {
	return tea.Batch(
		m.blastRadiusTask.Init(),
		m.riskTask.Init(),
	)
}

func (m submitPlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case submitPlanNowMsg:
		m.blastRadiusTask.status = taskStatusRunning
		m.blastRadiusTask.state = "executed terraform plan"

		cmds = append(cmds,
			m.submitPlanCmd,
			m.blastRadiusTask.spinner.Tick,
			m.waitForSubmitPlanActivity,
		)

	case submitPlanUpdateMsg:
		m.blastRadiusTask.state = "processing"
		m.progress = append(m.progress, msg.text)
		cmds = append(cmds, m.waitForSubmitPlanActivity)
	case submitPlanFinishedMsg:
		m.blastRadiusTask.status = taskStatusDone
		m.blastRadiusTask.state = "finished"
		m.riskTask.status = taskStatusDone
		m.progress = append(m.progress, msg.text)
		cmds = append(cmds, m.waitForSubmitPlanActivity)
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

		m.blastRadiusTask.status = taskStatusDone
		m.blastRadiusTask.state = "Change updated"
		cmds = append(cmds, m.waitForSubmitPlanActivity)

	case startSnapshotMsg:
		cmds = append(cmds, m.waitForSubmitPlanActivity)
	case progressSnapshotMsg:
		m.blastRadiusItems = msg.items
		m.blastRadiusEdges = msg.edges
		cmds = append(cmds, m.waitForSubmitPlanActivity)
	case finishSnapshotMsg:
		m.blastRadiusItems = msg.items
		m.blastRadiusEdges = msg.edges

	default:
		// propagate commands to components
		var cmd tea.Cmd

		for i, ms := range m.riskMilestoneTasks {
			m.riskMilestoneTasks[i], cmd = ms.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	m.blastRadiusTask, cmd = m.blastRadiusTask.Update(msg)
	cmds = append(cmds, cmd)

	m.riskTask, cmd = m.riskTask.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m submitPlanModel) View() string {
	bits := []string{}

	if m.blastRadiusTask.status != taskStatusPending {
		bits = append(bits, m.blastRadiusTask.View())
	}

	if m.riskTask.status != taskStatusPending {
		bits = append(bits, m.riskTask.View())
	}

	if m.changeUrl != "" {
		for _, t := range m.riskMilestoneTasks {
			bits = append(bits, fmt.Sprintf("   %v", t.View()))
		}
		if len(m.risks) == 0 {
			bits = append(bits, fmt.Sprintf("\nCheck the blast radius graph at:\n%v\n\n", m.changeUrl))
		} else {
			bits = append(bits, fmt.Sprintf("\nCheck the blast radius graph and risks at:\n%v\n\n", m.changeUrl))
		}
	}

	return strings.Join(bits, "\n") + "\n"
}

func (m submitPlanModel) Status() taskStatus {
	if m.blastRadiusTask.status != taskStatusDone {
		return m.blastRadiusTask.status
	}
	return m.riskTask.status
}

// A command that waits for the activity on the processing channel.
func (m submitPlanModel) waitForSubmitPlanActivity() tea.Msg {
	msg := <-m.processing
	log.Debugf("waitForSubmitPlanActivity received %T: %+v", msg, msg)
	return msg
}

func (m submitPlanModel) submitPlanCmd() tea.Msg {
	ctx := m.ctx
	span := trace.SpanFromContext(ctx)

	m.processing <- m.blastRadiusTask.StartMsg("converting terraform plan to JSON")

	if viper.GetString("ovm-test-fake") != "" {
		m.processing <- submitPlanUpdateMsg{"Fake processing json plan"}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{"Fake creating a new change"}
		time.Sleep(time.Second)
		m.processing <- m.blastRadiusTask.ProgressMsg("fake processing", 1, 2)
		time.Sleep(time.Second)
		m.processing <- changeUpdatedMsg{url: "https://example.com/changes/abc"}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{"Fake CalculateBlastRadiusResponse Status update: progress"}
		time.Sleep(time.Second)

		m.processing <- m.blastRadiusTask.ProgressMsg("discovering blast radius", 10, 21)
		time.Sleep(time.Second)

		m.processing <- changeUpdatedMsg{url: "https://example.com/changes/abc"}
		m.processing <- submitPlanUpdateMsg{"Calculating risks"}
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

		m.processing <- submitPlanFinishedMsg{"Fake done"}
		time.Sleep(time.Second)
		m.processing <- m.blastRadiusTask.FinishMsg("fake done", 100, 200)
		time.Sleep(time.Second)
		return nil
	}

	tfPlanJsonCmd := exec.CommandContext(ctx, "terraform", "show", "-json", m.planFile) // nolint:gosec // this is the file `terraform plan` already wrote to, so it's safe enough

	tfPlanJsonCmd.Stderr = os.Stderr // TODO: capture and output this through the View() instead

	planJson, err := tfPlanJsonCmd.Output()
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed to convert terraform plan to JSON: %w", err)}
	}

	plannedChanges, err := mappedItemDiffsFromPlan(ctx, planJson, m.planFile, log.Fields{})
	if err != nil {
		close(m.processing)
		return fatalError{err: fmt.Errorf("processPlanCmd: failed to parse terraform plan: %w", err)}
	}

	m.processing <- submitPlanUpdateMsg{"converted terraform plan to JSON"}
	m.processing <- m.blastRadiusTask.ProgressMsg("converted terraform plan to JSON", 0, 0)

	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		ticketLink, err = getTicketLinkFromPlan(m.planFile)
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
		m.processing <- submitPlanUpdateMsg{"Creating a new change"}
		m.processing <- m.blastRadiusTask.ProgressMsg("creating a new change", 0, 0)
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
		m.processing <- submitPlanUpdateMsg{"Updating an existing change"}
		m.processing <- m.blastRadiusTask.ProgressMsg("updating an existing change", 0, 0)
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

	m.processing <- submitPlanUpdateMsg{"Uploading planned changes"}
	log.WithField("change", changeUuid).Debug("Uploading planned changes")
	m.processing <- m.blastRadiusTask.ProgressMsg("uploading planned changes", 0, 0)

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
		m.processing <- submitPlanUpdateMsg{fmt.Sprintf("Status update: %v", msg)}
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
		m.processing <- m.blastRadiusTask.ProgressMsg(stateLabel, msg.GetNumItems(), msg.GetNumEdges())
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
	m.processing <- submitPlanUpdateMsg{"Calculating risks"}
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

	m.processing <- submitPlanFinishedMsg{"Done"}
	m.processing <- m.blastRadiusTask.FinishMsg("calculated blast radius and risks", msg.GetNumItems(), msg.GetNumEdges())

	return nil
}

func (m submitPlanModel) FinalReport() string {
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
