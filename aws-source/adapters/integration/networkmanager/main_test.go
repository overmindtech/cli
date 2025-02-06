package networkmanager

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	awsnetworkmanager "github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func TestMain(m *testing.M) {
	if integration.ShouldRunIntegrationTests() {
		fmt.Println("Running integration tests")
		os.Exit(m.Run())
	} else {
		fmt.Println("Skipping integration tests, set RUN_INTEGRATION_TESTS=true to run them")
		os.Exit(0)
	}
}

func TestIntegrationNetworkManager(t *testing.T) {
	t.Run("Setup", Setup)
	t.Run("NetworkManager", NetworkManager)
	t.Run("Teardown", Teardown)
}

func Setup(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := networkManagerClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create NetworkManager client: %v", err)
	}

	if err := setup(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to setup NetworkManager integration tests: %v", err)
	}
}

func Teardown(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := networkManagerClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create NetworkManager client: %v", err)
	}

	if err := teardown(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to teardown NetworkManager integration tests: %v", err)
	}
}

func networkManagerClient(ctx context.Context) (*awsnetworkmanager.Client, error) {
	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS settings: %w", err)
	}

	return awsnetworkmanager.NewFromConfig(testAWSConfig.Config), nil
}
