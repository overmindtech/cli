package ec2transitgateway

import (
	"context"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/go/sdpcache"
)

// TransitGatewayRouteTablePropagation runs the integration test for the route table propagation adapter.
// Setup creates a TGW VPC attachment (propagated to the default route table), so we get at least one propagation.
func TransitGatewayRouteTablePropagation(t *testing.T) {
	ctx := context.Background()

	testClient, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		t.Fatalf("Failed to get AWS settings: %v", err)
	}

	scope := adapters.FormatScope(testAWSConfig.AccountID, testAWSConfig.Region)
	adapter := adapters.NewEC2TransitGatewayRouteTablePropagationAdapter(testClient, testAWSConfig.AccountID, testAWSConfig.Region, sdpcache.NewNoOpCache())

	if err := adapter.Validate(); err != nil {
		t.Fatalf("failed to validate adapter: %v", err)
	}

	items, err := adapter.List(ctx, scope, true)
	if err != nil {
		t.Fatalf("failed to list transit gateway route table propagations: %v", err)
	}

	if len(items) == 0 {
		t.Fatalf("expected at least one propagation (Setup creates a TGW VPC attachment); got 0")
	}

	query := items[0].UniqueAttributeValue()
	got, err := adapter.Get(ctx, scope, query, true)
	if err != nil {
		t.Fatalf("failed to get propagation %s: %v", query, err)
	}
	if got.UniqueAttributeValue() != query {
		t.Fatalf("expected %s, got %s", query, got.UniqueAttributeValue())
	}

	// Search by route table ID (used by route table → propagation link).
	if createdRouteTableID != "" {
		searchItems, err := adapter.Search(ctx, scope, createdRouteTableID, true)
		if err != nil {
			t.Fatalf("failed to search propagations by route table ID %s: %v", createdRouteTableID, err)
		}
		if len(searchItems) == 0 {
			t.Fatalf("expected at least one propagation for route table %s (Setup creates one); got 0", createdRouteTableID)
		}
	}
}
