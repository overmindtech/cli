package ec2transitgateway

import (
	"context"
	"fmt"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/go/sdpcache"
)

// TransitGatewayRouteTable runs the integration test for the transit gateway route table adapter.
//
// AWS CLI – list route tables (same data this test lists/gets/searches):
//
//	aws ec2 describe-transit-gateway-route-tables [--region REGION]
//
// AWS Console – Transit Gateway route tables:
//
//	https://[REGION].console.aws.amazon.com/ec2/home?region=[REGION]#TransitGatewayRouteTables:
//
// Overmind – In the app, open your AWS source and search for type ec2-transit-gateway-route-table
// or navigate to the resource type in the source.
func TransitGatewayRouteTable(t *testing.T) {
	ctx := context.Background()

	testClient, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		t.Fatalf("Failed to get AWS settings: %v", err)
	}

	accountID := testAWSConfig.AccountID
	scope := adapters.FormatScope(accountID, testAWSConfig.Region)

	adapter := adapters.NewEC2TransitGatewayRouteTableAdapter(testClient, accountID, testAWSConfig.Region, sdpcache.NewNoOpCache())

	if err := adapter.Validate(); err != nil {
		t.Fatalf("failed to validate transit gateway route table adapter: %v", err)
	}

	items, err := listSync(adapter, ctx, scope, true)
	if err != nil {
		t.Fatalf("failed to list transit gateway route tables: %v", err)
	}

	tgwID, err := getIntegrationTestTransitGatewayID(ctx, testClient)
	if err != nil {
		t.Fatalf("failed to get integration-test transit gateway ID: %v", err)
	}

	// Find the route table for the transit gateway created in Setup (or discovered by tag).
	var routeTableID string
	for _, item := range items {
		tgwIDVal, _ := item.GetAttributes().Get("TransitGatewayId")
		if tgwIDVal != nil {
			if id, ok := tgwIDVal.(string); ok && id == tgwID {
				routeTableID = item.UniqueAttributeValue()
				break
			}
		}
	}
	if routeTableID == "" {
		t.Fatalf("no route table found for transit gateway %s (created in Setup)", tgwID)
	}

	got, err := adapter.Get(ctx, scope, routeTableID, true)
	if err != nil {
		t.Fatalf("failed to get transit gateway route table %s: %v", routeTableID, err)
	}

	if got.UniqueAttributeValue() != routeTableID {
		t.Fatalf("expected route table ID %s from Get, got %s", routeTableID, got.UniqueAttributeValue())
	}

	arn := fmt.Sprintf("arn:aws:ec2:%s:%s:transit-gateway-route-table/%s", testAWSConfig.Region, accountID, routeTableID)
	searchItems, err := searchSync(adapter, ctx, scope, arn, true)
	if err != nil {
		t.Fatalf("failed to search transit gateway route table by ARN: %v", err)
	}

	if len(searchItems) == 0 {
		t.Fatalf("search by ARN returned no items")
	}

	if searchItems[0].UniqueAttributeValue() != routeTableID {
		t.Fatalf("expected route table ID %s from Search, got %s", routeTableID, searchItems[0].UniqueAttributeValue())
	}

	// Route table links to associations, propagations, and routes (Search by route table ID).
	links := got.GetLinkedItemQueries()
	if len(links) < 4 {
		t.Fatalf("expected at least 4 linked item queries (ec2-transit-gateway + 3 Search links); got %d", len(links))
	}
	linkTypes := make(map[string]bool)
	for _, l := range links {
		if l.GetQuery() != nil {
			linkTypes[l.GetQuery().GetType()] = true
		}
	}
	for _, want := range []string{"ec2-transit-gateway", "ec2-transit-gateway-route-table-association", "ec2-transit-gateway-route-table-propagation", "ec2-transit-gateway-route"} {
		if !linkTypes[want] {
			t.Errorf("expected route table to link to %s", want)
		}
	}
}
