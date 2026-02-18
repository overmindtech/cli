package ec2transitgateway

import (
	"context"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/go/sdpcache"
)

// TransitGatewayRoute runs the integration test for the transit gateway route adapter.
// Setup creates a static route in the default route table, so we get at least one route.
func TransitGatewayRoute(t *testing.T) {
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
	adapter := adapters.NewEC2TransitGatewayRouteAdapter(testClient, testAWSConfig.AccountID, testAWSConfig.Region, sdpcache.NewNoOpCache())

	if err := adapter.Validate(); err != nil {
		t.Fatalf("failed to validate adapter: %v", err)
	}

	items, err := adapter.List(ctx, scope, true)
	if err != nil {
		t.Fatalf("failed to list transit gateway routes: %v", err)
	}

	if len(items) == 0 {
		t.Fatalf("expected at least one route (Setup creates a static TGW route); got 0")
	}

	query := items[0].UniqueAttributeValue()
	got, err := adapter.Get(ctx, scope, query, true)
	if err != nil {
		t.Fatalf("failed to get route %s: %v", query, err)
	}
	if got.UniqueAttributeValue() != query {
		t.Fatalf("expected %s, got %s", query, got.UniqueAttributeValue())
	}

	// Search by route table ID (used by route table → route link).
	if createdRouteTableID != "" {
		searchItems, err := adapter.Search(ctx, scope, createdRouteTableID, true)
		if err != nil {
			t.Fatalf("failed to search routes by route table ID %s: %v", createdRouteTableID, err)
		}
		if len(searchItems) == 0 {
			t.Fatalf("expected at least one route for route table %s (Setup creates a static route); got 0", createdRouteTableID)
		}
	}
}
