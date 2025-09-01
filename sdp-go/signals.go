package sdp

type SignalCategoryName string

// SignalCategoryName constants represent the predefined categories for signals.
// if you add a new category, please also update the cli command "submit-signal" @ cli/cmd/changes_submit_signal.go
const (
	SignalCategoryNameCustom  SignalCategoryName = "Custom"
	SignalCategoryNameRoutine SignalCategoryName = "Routine"
)
