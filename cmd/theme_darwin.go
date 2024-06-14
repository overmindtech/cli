package cmd

// IsConhost returns true if the current terminal is conhost. This indicates
// that it can't deal with multi-byte characters and requires special treatment.
// See https://github.com/overmindtech/cli/issues/388 for detailed analysis.
func IsConhost() bool {
	return false
}
