package shared_test

import (
	_ "github.com/overmindtech/cli/sources/gcp/dynamic/adapters" // Import all adapters to register them
)

// This file ensures that all adapters are registered before running tests in the shared package.
// The package is "shared_test" (not "shared") to avoid import cycle issues.
