//go:build integration

package adapters

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

// TestCloudwatchInstanceMetricIntegration fetches real CloudWatch metrics for an EC2 instance
// Run with: TEST_INSTANCE_ID=i-xxx AWS_PROFILE=terraform-example go test -v -tags=integration -run "TestCloudwatchInstanceMetricIntegration" ./aws-source/adapters/...
func TestCloudwatchInstanceMetricIntegration(t *testing.T) {
	instanceID := os.Getenv("TEST_INSTANCE_ID")
	if instanceID == "" {
		t.Skip("Skipping integration test: TEST_INSTANCE_ID environment variable not set")
	}

	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := cloudwatch.NewFromConfig(config)

	adapter := NewCloudwatchInstanceMetricAdapter(client, account, region, nil)
	scope := adapterhelpers.FormatScope(account, region)

	// Query is just the instance ID
	query := instanceID

	t.Logf("Querying CloudWatch for instance: %s", instanceID)
	t.Logf("Query: %s", query)

	ctx := context.Background()

	item, err := adapter.Get(ctx, scope, query, false)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	// Pretty print the item attributes
	attrs := item.GetAttributes().GetAttrStruct().AsMap()
	prettyJSON, _ := json.MarshalIndent(attrs, "", "  ")

	t.Logf("\n=== CloudWatch Instance Metric Result ===\n%s\n", string(prettyJSON))

	// Log key metrics
	t.Logf("\n=== Summary ===")
	t.Logf("Instance: %s", instanceID)
	t.Logf("Data Available: %v", attrs["DataAvailable"])
	if attrs["DataAvailable"] == true {
		t.Logf("Last Updated: %v", attrs["LastUpdated"])
		// Log all metrics
		for _, metricName := range ec2InstanceMetrics {
			if value, exists := attrs[metricName]; exists {
				t.Logf("%s: %v", metricName, value)
			}
		}
	} else {
		t.Logf("No data available for this instance")
	}
}
