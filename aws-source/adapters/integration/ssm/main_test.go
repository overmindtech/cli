package ssm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func TestMain(m *testing.M) {
	if integration.ShouldRunIntegrationTests() {
		fmt.Println("Running SSM integration tests")
		exitCode := func() int {
			defer tracing.ShutdownTracer(context.Background())

			if err := tracing.InitTracerWithUpstreams("ssm-integration-tests", os.Getenv("HONEYCOMB_API_KEY"), ""); err != nil {
				log.Fatal(err)
			}

			return m.Run()
		}()

		os.Exit(exitCode)
	} else {
		fmt.Println("Skipping SSM integration tests")
		os.Exit(0)
	}
}

var tracer = otel.GetTracerProvider().Tracer(
	"SSMIntegrationTests",
)

func TestIntegrationSSM(t *testing.T) {
	t.Run("Setup", func(t *testing.T) {
		ctx := context.Background()

		testAWSConfig, err := integration.AWSSettings(ctx)
		if err != nil {
			t.Fatalf("Failed to get AWS settings: %v", err)
		}

		client := ssm.NewFromConfig(testAWSConfig.Config)
		testID := integration.TestID()

		// Define hierarchy levels
		environments := []string{"prod", "stage"}
		regions := []string{"us-east-1", "eu-west-1"}
		services := []string{"api", "web", "worker"}
		components := []string{"database", "cache"}
		configs := []string{"connection", "auth", "monitoring"}

		// Create parameters with balanced hierarchy
		for _, env := range environments {
			for _, region := range regions {
				for _, service := range services {
					for _, component := range components {
						for _, config := range configs {
							for i := range 1 {
								path := fmt.Sprintf("/integration-test/%s/%s/%s/%s/%s/param%d",
									env, region, service, component, config, i)

								_, err = client.PutParameter(ctx, &ssm.PutParameterInput{
									Name:  aws.String(path),
									Type:  types.ParameterTypeString,
									Value: aws.String(fmt.Sprintf("test-value-%s-%d", config, i)),
									Tags: []types.Tag{
										{
											Key:   aws.String(integration.TagTestKey),
											Value: aws.String(integration.TagTestValue),
										},
										{
											Key:   aws.String(integration.TagTestIDKey),
											Value: aws.String(testID),
										},
									},
								})
								if err != nil {
									var alreadyExistsErr *types.ParameterAlreadyExists
									if errors.As(err, &alreadyExistsErr) {
										// Skip if parameter already exists
										continue
									} else {
										t.Fatalf("Failed to create test parameter %s: %v", path, err)
									}
								}
							}
							// Log progress for each leaf node completion
							t.Logf("Created parameters for %s/%s/%s/%s/%s", env, region, service, component, config)
						}
					}
				}
			}
		}

		t.Log("Successfully created all test parameters")
	})

	t.Run("SSM", func(t *testing.T) {
		ctx := context.Background()

		testAWSConfig, err := integration.AWSSettings(ctx)
		if err != nil {
			t.Fatalf("Failed to get AWS settings: %v", err)
		}

		client := ssm.NewFromConfig(testAWSConfig.Config)
		scope := testAWSConfig.AccountID + "." + testAWSConfig.Region

		adapter := adapters.NewSSMParameterAdapter(client, testAWSConfig.AccountID, testAWSConfig.Region)

		ctx, span := tracer.Start(ctx, "SSM.List")
		defer span.End()
		start := time.Now()

		stream := discovery.NewRecordingQueryResultStream()
		adapter.ListStream(ctx, scope, false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		t.Logf("Listed %d SSM parameters in %v", len(items), time.Since(start))

		span.SetAttributes(
			attribute.Int("ssm.parameters", len(items)),
		)
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := context.Background()

		testAWSConfig, err := integration.AWSSettings(ctx)
		if err != nil {
			t.Fatalf("Failed to get AWS settings: %v", err)
		}

		client := ssm.NewFromConfig(testAWSConfig.Config)
		testID := integration.TestID()

		var nextToken *string
		deleted := 0

		for {
			// Get parameters by path recursively
			input := &ssm.GetParametersByPathInput{
				Path:      aws.String("/integration-test"),
				Recursive: aws.Bool(true),
				NextToken: nextToken,
				ParameterFilters: []types.ParameterStringFilter{
					{
						Key: aws.String("tag:" + integration.TagTestIDKey),
						Values: []string{
							testID,
						},
					},
				},
			}

			output, err := client.GetParametersByPath(ctx, input)
			if err != nil {
				t.Fatalf("Failed to get parameters for deletion: %v", err)
			}

			if len(output.Parameters) == 0 {
				break
			}

			// Delete parameters in batches of 100
			for i := 0; i < len(output.Parameters); i += 100 {
				end := i + 100
				if end > len(output.Parameters) {
					end = len(output.Parameters)
				}

				batch := output.Parameters[i:end]
				names := make([]string, len(batch))
				for j, param := range batch {
					names[j] = *param.Name
				}

				_, err := client.DeleteParameters(ctx, &ssm.DeleteParametersInput{
					Names: names,
				})
				if err != nil {
					t.Fatalf("Failed to delete parameters: %v", err)
				}

				deleted += len(names)
				t.Logf("Deleted %d parameters...", deleted)
			}

			if output.NextToken == nil {
				break
			}
			nextToken = output.NextToken
		}

		t.Logf("Successfully deleted %d test parameters", deleted)
	})
}
