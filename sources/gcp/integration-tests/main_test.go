package integrationtests

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

func TestMain(m *testing.M) {
	if shouldRunIntegrationTests() {
		fmt.Println("Running integration tests")
		os.Exit(m.Run())
	} else {
		fmt.Println("Skipping integration tests, set RUN_GCP_INTEGRATION_TESTS=true to run them")
		os.Exit(0)
	}
}

func shouldRunIntegrationTests() bool {
	run, found := os.LookupEnv("RUN_GCP_INTEGRATION_TESTS")

	if !found {
		return false
	}

	shouldRun, err := strconv.ParseBool(run)
	if err != nil {
		return false
	}

	return shouldRun
}
