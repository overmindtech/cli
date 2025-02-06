package kms

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
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

func TestIntegrationKMS(t *testing.T) {
	t.Run("Setup", Setup)
	t.Run("KMS", KMS)
	t.Run("Teardown", Teardown)
}

func Setup(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := kmsClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create KMS client: %v", err)
	}

	if err := setup(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to setup KMS integration tests: %v", err)
	}
}

func Teardown(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := kmsClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create KMS client: %v", err)
	}

	if err := teardown(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to teardown KMS integration tests: %v", err)
	}
}

func kmsClient(ctx context.Context) (*awskms.Client, error) {
	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS settings: %w", err)
	}

	return awskms.NewFromConfig(testAWSConfig.Config), nil
}
