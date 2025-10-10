package dynamic_test

import (
	"context"
	"testing"

	_ "github.com/overmindtech/cli/sources/gcp/dynamic/adapters" // Import all adapters to register them
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// TODO: Possible improvements:
// - Create a helper function that does some of the common assertions for the adapter tests
func TestAdapter(t *testing.T) {
	_ = context.Background()
	_ = "test-project"
	_ = gcpshared.NewLinker()

	// All adapter tests have been moved to individual test files
	// This file now only serves to import all adapters to register them
}
