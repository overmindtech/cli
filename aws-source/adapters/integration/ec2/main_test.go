package ec2

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
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

func TestIntegrationEC2(t *testing.T) {
	t.Run("Setup", Setup)
	t.Run("EC2", EC2)
	t.Run("Teardown", Teardown)
}

func Setup(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	if err := setup(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to setup EC2 integration tests: %v", err)
	}
}

func Teardown(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	if err := teardown(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to teardown EC2 integration tests: %v", err)
	}
}

func ec2Client(ctx context.Context) (*awsec2.Client, error) {
	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS settings: %w", err)
	}

	return awsec2.NewFromConfig(testAWSConfig.Config), nil
}
