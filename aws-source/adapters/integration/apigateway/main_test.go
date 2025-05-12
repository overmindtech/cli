package apigateway

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func TestMain(m *testing.M) {
	if integration.ShouldRunIntegrationTests() {
		fmt.Println("Running apigateway integration tests")
		os.Exit(m.Run())
	} else {
		fmt.Println("Skipping apigateway integration tests, set RUN_INTEGRATION_TESTS=true to run them")
		os.Exit(0)
	}
}

func TestIntegrationAPIGateway(t *testing.T) {
	t.Run("Setup", Setup)
	t.Run("APIGateway", APIGateway)
	t.Run("Teardown", Teardown)
}

func Setup(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := apigatewayClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create APIGateway client: %v", err)
	}

	if err := setup(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to setup APIGateway integration tests: %v", err)
	}
}

func Teardown(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	var err error
	testClient, err := apigatewayClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create APIGateway client: %v", err)
	}

	if err := teardown(ctx, logger, testClient); err != nil {
		t.Fatalf("Failed to teardown APIGateway integration tests: %v", err)
	}
}

func apigatewayClient(ctx context.Context) (*apigateway.Client, error) {
	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	return apigateway.NewFromConfig(testAWSConfig.Config), nil
}
