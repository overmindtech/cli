package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
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

	// this channel transports updates from the submitPlan processing. The
	// submitPlanUpdateMsg is a wrapper around the actual tea.Msg to make it
	// simpler to pump the waitForSubmitPlanActivity command
	processing chan submitPlanUpdateMsg
	progress   []string
	changeUrl  string

	removingSecretsTask    taskModel
	resourceExtractionTask taskModel
	planMappingResult      PlanMappingResult

	uploadChangesTask taskModel

	blastRadiusTask  snapshotModel
	blastRadiusItems uint32
	blastRadiusEdges uint32

	riskTask           taskModel
	risksStarted       time.Time
	riskMilestones     []*sdp.RiskCalculationStatus_ProgressMilestone
	riskMilestoneTasks []taskModel
	risks              []*sdp.Risk

	width int
}
type submitPlanNowMsg struct{}

type submitPlanUpdateMsg struct{ wrapped tea.Msg }
type submitPlanFinishedMsg struct{ text string }

type changeUpdatedMsg struct {
	url            string
	riskMilestones []*sdp.RiskCalculationStatus_ProgressMilestone
	risks          []*sdp.Risk
}

func NewSubmitPlanModel(planFile string, width int) submitPlanModel {
	return submitPlanModel{
		planFile: planFile,

		processing: make(chan submitPlanUpdateMsg, 1000), // provide a buffer for sending updates, so we don't block the processing
		progress:   []string{},

		removingSecretsTask:    NewTaskModel("Removing secrets", width),
		resourceExtractionTask: NewTaskModel("Extracting resources", width),
		uploadChangesTask:      NewTaskModel("Uploading planned changes", width),

		blastRadiusTask: NewSnapShotModel("Calculating Blast Radius", "Discovering dependencies", width),
		riskTask:        NewTaskModel("Calculating Risks", width),
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
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case submitPlanNowMsg:
		cmds = append(cmds,
			m.submitPlanCmd,
			m.waitForSubmitPlanActivity,
		)

		if os.Getenv("CI") == "" {
			cmds = append(cmds,
				m.removingSecretsTask.spinner.Tick,
				m.resourceExtractionTask.spinner.Tick,
				m.uploadChangesTask.spinner.Tick,
			)
		}

	case submitPlanUpdateMsg:
		// ensure that the wrapped message is submitted before we wait for the
		// next update. This is still not perfect, but there's currently no
		// better idea on the table.
		cmds = append(cmds, tea.Sequence(func() tea.Msg { return msg.wrapped }, m.waitForSubmitPlanActivity))

	case *PlanMappingResult:
		m.planMappingResult = *msg

	case submitPlanFinishedMsg:
		m.riskTask.status = taskStatusDone
		m.progress = append(m.progress, msg.text)
	case changeUpdatedMsg:
		m.changeUrl = msg.url
		m.riskMilestones = msg.riskMilestones
		if len(m.riskMilestoneTasks) != len(msg.riskMilestones) {
			m.riskMilestoneTasks = []taskModel{}
			for _, ms := range msg.riskMilestones {
				tm := NewTaskModel(ms.GetDescription(), m.width)
				tm.indent = 4
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
				if os.Getenv("CI") == "" {
					cmds = append(cmds, m.riskMilestoneTasks[i].spinner.Tick)
				}
			case sdp.RiskCalculationStatus_ProgressMilestone_STATUS_SKIPPED:
				m.riskMilestoneTasks[i].status = taskStatusSkipped
			}
		}
		m.risks = msg.risks

		if len(m.riskMilestones) > 0 {
			m.riskTask.status = taskStatusRunning
			if os.Getenv("CI") == "" {
				cmds = append(cmds, m.riskTask.spinner.Tick)
			}

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

			if m.risksStarted == (time.Time{}) {
				m.risksStarted = time.Now()
			}
		} else if len(m.risks) > 0 {
			m.riskTask.status = taskStatusDone
		}

	case progressSnapshotMsg:
		m.blastRadiusItems = msg.items
		m.blastRadiusEdges = msg.edges

	}

	var cmd tea.Cmd
	m.removingSecretsTask, cmd = m.removingSecretsTask.Update(msg)
	cmds = append(cmds, cmd)

	m.resourceExtractionTask, cmd = m.resourceExtractionTask.Update(msg)
	cmds = append(cmds, cmd)

	m.uploadChangesTask, cmd = m.uploadChangesTask.Update(msg)
	cmds = append(cmds, cmd)

	m.blastRadiusTask, cmd = m.blastRadiusTask.Update(msg)
	cmds = append(cmds, cmd)

	m.riskTask, cmd = m.riskTask.Update(msg)
	cmds = append(cmds, cmd)

	for i, ms := range m.riskMilestoneTasks {
		m.riskMilestoneTasks[i], cmd = ms.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m submitPlanModel) View() string {
	bits := []string{}

	if m.removingSecretsTask.status != taskStatusPending {
		bits = append(bits, m.removingSecretsTask.View())
	}

	if m.resourceExtractionTask.status != taskStatusPending {
		bits = append(bits, m.resourceExtractionTask.View())
		for _, mapping := range m.planMappingResult.Results {
			var icon string
			switch mapping.Status {
			case MapStatusSuccess:
				icon = RenderOk()
			case MapStatusNotEnoughInfo:
				icon = RenderUnknown()
			case MapStatusUnsupported:
				icon = RenderErr()
			}
			bits = append(bits, fmt.Sprintf("  %v %v (%v)", icon, mapping.TerraformName, mapping.Message))
		}
	}

	if m.uploadChangesTask.status != taskStatusPending {
		bits = append(bits, m.uploadChangesTask.View())
	}

	if m.blastRadiusTask.overall.status != taskStatusPending {
		bits = append(bits, m.blastRadiusTask.View())
	}

	if m.riskTask.status != taskStatusPending {
		bits = append(bits, m.riskTask.View())
		for _, t := range m.riskMilestoneTasks {
			bits = append(bits, fmt.Sprintf("  %v", t.View()))
		}
	}

	if m.changeUrl != "" && m.riskTask.status != taskStatusDone && m.riskTask.status != taskStatusError && time.Since(m.risksStarted) > 1500*time.Millisecond {
		bits = append(bits, fmt.Sprintf("  │ Check the blast radius graph while you wait:\n  │ %v\n", m.changeUrl))
	}

	return strings.Join(bits, "\n") + "\n"
}

func (m submitPlanModel) Status() taskStatus {
	// return taskStatusPending when the first task is still pending
	if m.removingSecretsTask.status != taskStatusDone {
		return m.removingSecretsTask.status
	}

	if m.removingSecretsTask.status != taskStatusPending && m.resourceExtractionTask.status != taskStatusDone {
		return m.resourceExtractionTask.status
	}

	if m.uploadChangesTask.status != taskStatusPending && m.uploadChangesTask.status != taskStatusDone {
		return m.uploadChangesTask.status
	}

	if m.blastRadiusTask.overall.status != taskStatusPending && m.blastRadiusTask.overall.status != taskStatusDone {
		return m.blastRadiusTask.overall.status
	}

	// return taskStatusDone when the last task is done
	if m.riskTask.status != taskStatusPending {
		return m.riskTask.status
	}

	// return taskStatusRunning when no task has errored or skipped
	return taskStatusRunning
}

// A command that waits for the activity on the processing channel.
func (m submitPlanModel) waitForSubmitPlanActivity() tea.Msg {
	return <-m.processing
}

func (m submitPlanModel) risksError(msg string, err error) tea.Msg {
	if m.changeUrl == "" {
		if err == nil {
			return fatalError{err: errors.New(msg)}
		}
		return fatalError{err: fmt.Errorf("%v: %w", msg, err)}
	} else {
		if err == nil {
			return fatalError{err: fmt.Errorf("%v\nWe'll retry in the background. Find the results online when they're done: %v", msg, m.changeUrl)}
		}
		return fatalError{err: fmt.Errorf("%v: %w\nWe'll retry in the background. Find the results online when they're done: %v", msg, err, m.changeUrl)}
	}
}

func (m submitPlanModel) submitPlanCmd() tea.Msg {
	ctx := m.ctx
	span := trace.SpanFromContext(ctx)

	if viper.GetString("ovm-test-fake") != "" {
		m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusRunning)}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusDone)}
		m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateStatusMsg(taskStatusRunning)}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateTitleMsg(
			"Extracting 13 changing resources: 4 supported 9 unsupported",
		)}
		mappingResult := PlanMappingResult{
			Results: []PlannedChangeMapResult{
				{
					TerraformName:  "kubernetes_deployment.nats_box",
					Status:         MapStatusSuccess,
					Message:        "mapped",
					MappedItemDiff: &sdp.MappedItemDiff{},
				},
				{
					TerraformName:  "kubernetes_deployment.api_server",
					Status:         MapStatusNotEnoughInfo,
					Message:        "missing arn",
					MappedItemDiff: &sdp.MappedItemDiff{},
				},
				{
					TerraformName:  "aws_fake_resource",
					Status:         MapStatusUnsupported,
					Message:        "unsupported",
					MappedItemDiff: &sdp.MappedItemDiff{},
				},
			},
			RemovedSecrets: 12,
		}
		m.processing <- submitPlanUpdateMsg{&mappingResult}

		m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateStatusMsg(taskStatusDone)}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusRunning)}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateTitleMsg("Uploading planned changes (new/existing)")}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusDone)}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.StartMsg()}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.ProgressMsg("fake processing", 1, 2)}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.ProgressMsg("fake processing", 3, 4)}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.SavingMsg()}
		time.Sleep(time.Second)
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.FinishMsg()}

		// update local copy
		m.changeUrl = "https://example.com/changes/abc"
		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{url: "https://example.com/changes/abc"}}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusRunning)}
		time.Sleep(100 * time.Millisecond)
		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
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
		}}
		time.Sleep(1500 * time.Millisecond)

		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
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
		}}
		time.Sleep(1500 * time.Millisecond)

		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
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
		}}
		time.Sleep(1500 * time.Millisecond)

		if viper.GetString("ovm-test-fake") == "risks-error" {
			m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
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
						Description: "fake errored milestone",
						Status:      sdp.RiskCalculationStatus_ProgressMilestone_STATUS_ERROR,
					},
				},
				risks: []*sdp.Risk{},
			}}

			m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("initial risk calculation errored", nil)}
			close(m.processing)
			return nil
		}
		high := uuid.New()
		medium := uuid.New()
		low := uuid.New()
		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
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
		}}
		time.Sleep(time.Second)

		m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusDone)}
		m.processing <- submitPlanUpdateMsg{submitPlanFinishedMsg{"Fake done"}}
		time.Sleep(time.Second)
		return nil
	}

	///////////////////////////////////////////////////////////////////
	// Convert provided plan into JSON for easier parsing
	///////////////////////////////////////////////////////////////////
	m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusRunning)}
	tfPlanJsonCmd := exec.CommandContext(ctx, "terraform", "show", "-json", m.planFile) // nolint:gosec // this is the file `terraform plan` already wrote to, so it's safe enough

	tfPlanJsonCmd.Stderr = os.Stderr // TODO: capture and output this through the View() instead

	log.WithField("args", tfPlanJsonCmd.Args).Debug("converting plan to JSON")
	planJson, err := tfPlanJsonCmd.Output()
	if err != nil {
		m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("failed to convert terraform plan to JSON", err)}
		close(m.processing)
		return nil
	}

	m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusDone)}

	///////////////////////////////////////////////////////////////////
	// Extract changes from the plan and created mapped item diffs
	///////////////////////////////////////////////////////////////////
	m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateStatusMsg(taskStatusRunning)}
	time.Sleep(200 * time.Millisecond) // give the UI a little time to update

	// Map the terraform changes to Overmind queries
	mappingResponse, err := mappedItemDiffsFromPlan(ctx, planJson, m.planFile, log.Fields{})
	if err != nil {
		m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("failed to parse terraform plan", err)}
		close(m.processing)
		return nil
	}
	m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateTitleMsg(
		fmt.Sprintf("Removed %v secrets", mappingResponse.RemovedSecrets),
	)}

	m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateTitleMsg(
		fmt.Sprintf("Extracting %v changing resources: %v supported %v skipped %v unsupported",
			mappingResponse.NumTotal(),
			mappingResponse.NumSuccess(),
			mappingResponse.NumNotEnoughInfo(),
			mappingResponse.NumUnsupported(),
		))}

	// Sort the supported and unsupported changes so that they display nicely
	slices.SortFunc(mappingResponse.Results, func(a, b PlannedChangeMapResult) int {
		return int(a.Status) - int(b.Status)
	})
	m.processing <- submitPlanUpdateMsg{mappingResponse}
	time.Sleep(200 * time.Millisecond) // give the UI a little time to update
	m.processing <- submitPlanUpdateMsg{m.resourceExtractionTask.UpdateStatusMsg(taskStatusDone)}

	///////////////////////////////////////////////////////////////////
	// try to link up the plan with a Change and start submitting to the API
	///////////////////////////////////////////////////////////////////
	m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusRunning)}
	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		ticketLink, err = getTicketLinkFromPlan(m.planFile)
		if err != nil {
			m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("failed to get ticket link from plan", err)}
			close(m.processing)
			return nil
		}
	}

	client := AuthenticatedChangesClient(ctx, m.oi)
	changeUuid, err := getChangeUuid(ctx, m.oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, false)
	if err != nil {
		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("failed searching for existing changes", err)}
		close(m.processing)
		return nil
	}

	title := changeTitle(viper.GetString("title"))
	tfPlanTextCmd := exec.CommandContext(ctx, "terraform", "show", m.planFile) // nolint:gosec // this is the file `terraform plan` already wrote to, so it's safe enough

	tfPlanTextCmd.Stderr = os.Stderr // TODO: capture and output this through the View() instead

	log.WithField("args", tfPlanTextCmd.Args).Debug("converting plan to JSON")
	tfPlanOutput, err := tfPlanTextCmd.Output()
	if err != nil {
		m.processing <- submitPlanUpdateMsg{m.removingSecretsTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("failed to convert terraform plan to JSON", err)}
		close(m.processing)
		return nil
	}
	codeChangesOutput := tryLoadText(ctx, viper.GetString("code-changes-diff"))

	if changeUuid == uuid.Nil {
		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateTitleMsg("Uploading planned changes (new)")}
		log.Debug("Creating a new change")
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  ticketLink,
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
					RawPlan:     string(tfPlanOutput),
					CodeChanges: codeChangesOutput,
				},
			},
		})
		if err != nil {
			m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("failed to create a new change", err)}
			close(m.processing)
			return nil
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("failed to read change id", err)}
			close(m.processing)
			return nil
		}

		changeUuid = *maybeChangeUuid
		span.SetAttributes(
			attribute.String("ovm.change.uuid", changeUuid.String()),
			attribute.Bool("ovm.change.new", true),
		)
	} else {
		m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateTitleMsg("Uploading planned changes (update)")}
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
					RawPlan:     string(tfPlanOutput),
					CodeChanges: codeChangesOutput,
				},
			},
		})
		if err != nil {
			m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("failed to update change", err)}
			close(m.processing)
			return nil
		}
	}

	time.Sleep(200 * time.Millisecond) // give the UI a little time to update
	m.processing <- submitPlanUpdateMsg{m.uploadChangesTask.UpdateStatusMsg(taskStatusDone)}

	///////////////////////////////////////////////////////////////////
	// calculate blast radius and risks
	///////////////////////////////////////////////////////////////////
	m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.StartMsg()}
	log.WithField("change", changeUuid).Debug("Uploading planned changes")

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: mappingResponse.GetItemDiffs(),
		},
	})
	if err != nil {
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("failed to update planned changes", err)}
		close(m.processing)
		return nil
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
		stateLabel := "unknown"
		switch msg.GetState() {
		case sdp.CalculateBlastRadiusResponse_STATE_UNSPECIFIED:
			stateLabel = "unknown"
		case sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING:
			stateLabel = "discovering blast radius"
		case sdp.CalculateBlastRadiusResponse_STATE_FINDING_APPS:
			stateLabel = "finding apps"
		case sdp.CalculateBlastRadiusResponse_STATE_SAVING, sdp.CalculateBlastRadiusResponse_STATE_DONE:
			stateLabel = "done"
		}
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.ProgressMsg(stateLabel, msg.GetNumItems(), msg.GetNumEdges())}

		// send a message when the blast radius is saved
		if msg.GetState() == sdp.CalculateBlastRadiusResponse_STATE_SAVING {
			m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.SavingMsg()}
		}
	}
	if resultStream.Err() != nil {
		m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("error streaming results", err)}
		close(m.processing)
		return nil
	}
	m.processing <- submitPlanUpdateMsg{m.blastRadiusTask.FinishMsg()}

	changeUrl := *m.oi.FrontendUrl
	changeUrl.Path = fmt.Sprintf("%v/changes/%v/blast-radius", changeUrl.Path, changeUuid)
	log.WithField("change-url", changeUrl.String()).Info("Change ready")

	// update local copy
	m.changeUrl = changeUrl.String()
	m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{url: m.changeUrl}}

	///////////////////////////////////////////////////////////////////
	// wait for risk calculation to happen
	///////////////////////////////////////////////////////////////////
	m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusRunning)}
	risksErrored := false
	for {
		riskRes, err := client.GetChangeRisks(ctx, &connect.Request[sdp.GetChangeRisksRequest]{
			Msg: &sdp.GetChangeRisksRequest{
				UUID: changeUuid[:],
			},
		})
		if err != nil {
			m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("failed to get change risks", err)}
			close(m.processing)
			return nil
		}

		m.processing <- submitPlanUpdateMsg{changeUpdatedMsg{
			url:            m.changeUrl,
			riskMilestones: riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetProgressMilestones(),
			risks:          riskRes.Msg.GetChangeRiskMetadata().GetRisks(),
		}}

		status := riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetStatus()
		if status == sdp.RiskCalculationStatus_STATUS_UNSPECIFIED || status == sdp.RiskCalculationStatus_STATUS_INPROGRESS {
			time.Sleep(time.Second)
			// retry
		} else if status == sdp.RiskCalculationStatus_STATUS_ERROR {
			risksErrored = true
			break
		} else {
			// it's done
			break
		}

		if ctx.Err() != nil {
			err := ctx.Err()
			m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusError)}
			m.processing <- submitPlanUpdateMsg{m.risksError("context cancelled", err)}
			close(m.processing)
			return nil
		}
	}

	if risksErrored {
		m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusError)}
		m.processing <- submitPlanUpdateMsg{m.risksError("initial risk calculation errored", nil)}
		close(m.processing)
		return nil
	}

	m.processing <- submitPlanUpdateMsg{m.riskTask.UpdateStatusMsg(taskStatusDone)}
	m.processing <- submitPlanUpdateMsg{submitPlanFinishedMsg{"Done"}}

	return nil
}

func (m submitPlanModel) FinalReport() string {
	bits := []string{}
	if m.blastRadiusItems > 0 {
		bits = append(bits, styleH1().Render("Blast Radius"))
		bits = append(bits, fmt.Sprintf("\nItems: %v\nEdges: %v\n", m.blastRadiusItems, m.blastRadiusEdges))
	}
	if m.changeUrl != "" && len(m.risks) > 0 {
		bits = append(bits, styleH1().Render("Potential Risks"))
		bits = append(bits, "")
		for _, r := range m.risks {
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

			bits = append(bits, (fmt.Sprintf("%v%v\n\n%v\n\n",
				title,
				severity,
				wordwrap.String(r.GetDescription(), min(160, m.width-4)))))
		}
		bits = append(bits, fmt.Sprintf("\nCheck the blast radius graph and risks at:\n%v\n\n", m.changeUrl))
	}
	return strings.Join(bits, "\n") + "\n"
}
