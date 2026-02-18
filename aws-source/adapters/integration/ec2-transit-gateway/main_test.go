// Package ec2transitgateway runs integration tests for EC2 Transit Gateway adapters
// (transit gateway route table, route table association, route table propagation,
// and route). Setup creates a transit gateway, VPC, subnet, TGW VPC attachment,
// and a static route so each adapter returns items; Teardown deletes them in order.
//
// All created resources are tagged with name and test-id "integration-test" so they
// are easy to spot in the console and so Teardown can discover them by tag. You can
// run Setup once, re-run the test subtests as needed, then run Teardown once; or run
// Teardown alone to clean up any stale resources from a previous run.
//
// Run integration tests only when RUN_INTEGRATION_TESTS=true. Example CLI commands:
//
//	# Setup only (create resources)
//	RUN_INTEGRATION_TESTS=true go test ./aws-source/adapters/integration/ec2-transit-gateway -v -count=1 -run '^TestIntegrationEC2TransitGateway$/Setup$'
//
//	# Teardown only (delete resources by tag; idempotent)
//	RUN_INTEGRATION_TESTS=true go test ./aws-source/adapters/integration/ec2-transit-gateway -v -count=1 -run '^TestIntegrationEC2TransitGateway$/Teardown$'
//
//	# Run a single adapter test (e.g. after Setup, re-run as needed)
//	RUN_INTEGRATION_TESTS=true go test ./aws-source/adapters/integration/ec2-transit-gateway -v -count=1 -run '^TestIntegrationEC2TransitGateway$/TransitGatewayRouteTable$'
//
//	# Run the full suite (Setup, all adapter tests, Teardown)
//	RUN_INTEGRATION_TESTS=true go test ./aws-source/adapters/integration/ec2-transit-gateway -v -count=1 -run '^TestIntegrationEC2TransitGateway$'
//
// Cost: a few cents per run. Setup creates a Transit Gateway, a VPC, a subnet, and
// one TGW VPC attachment so that association, propagation, and route adapters
// return items. AWS charges for the TGW and ~$0.05/hour per VPC attachment; with
// teardown within minutes, cost remains low. See https://aws.amazon.com/transit-gateway/pricing/
//
// Per-adapter cost: route table, association, propagation, and route tests do not
// create additional resources; they list/get from the same TGW and its default
// route table (one attachment, one static route), so they add no extra cost.
//
// To inspect the infrastructure created by the tests:
//
//   - AWS CLI (replace [REGION] and [ROUTE_TABLE_ID] as needed):
//
//     aws ec2 describe-transit-gateways [--region [REGION]]
//     aws ec2 describe-transit-gateway-route-tables [--region [REGION]]
//     aws ec2 get-transit-gateway-route-table-associations --transit-gateway-route-table-id [ROUTE_TABLE_ID] [--region [REGION]]
//     aws ec2 get-transit-gateway-route-table-propagations --transit-gateway-route-table-id [ROUTE_TABLE_ID] [--region [REGION]]
//     aws ec2 search-transit-gateway-routes --transit-gateway-route-table-id [ROUTE_TABLE_ID] --filters "Name=state,Values=active,blackhole" [--region [REGION]]
//
//   - AWS Console: EC2 → Network & Security → Transit gateways → select a transit gateway
//     `https://eu-west-2.console.aws.amazon.com/vpcconsole/home?region=eu-west-2#TransitGateways:` other resources are displayed on the left hand pane.
package ec2transitgateway

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
)

func TestMain(m *testing.M) {
	if integration.ShouldRunIntegrationTests() {
		fmt.Println("Running EC2 Transit Gateway integration tests")
		os.Exit(m.Run())
	} else {
		fmt.Println("Skipping EC2 Transit Gateway integration tests, set RUN_INTEGRATION_TESTS=true to run them")
		os.Exit(0)
	}
}

func TestIntegrationEC2TransitGateway(t *testing.T) {
	// Setup creates resources tagged integration-test; Teardown is idempotent and discovers by tag.
	t.Run("Setup", Setup)
	t.Run("TransitGatewayRouteTable", TransitGatewayRouteTable)
	t.Run("TransitGatewayRouteTableAssociation", TransitGatewayRouteTableAssociation)
	t.Run("TransitGatewayRouteTablePropagation", TransitGatewayRouteTablePropagation)
	t.Run("TransitGatewayRoute", TransitGatewayRoute)
	t.Run("Teardown", Teardown)
}

func listSync(adapter discovery.ListStreamableAdapter, ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(ctx, scope, ignoreCache, stream)
	if errs := stream.GetErrors(); len(errs) > 0 {
		return nil, fmt.Errorf("failed to list: %v", errs)
	}
	return stream.GetItems(), nil
}

func searchSync(adapter discovery.SearchStreamableAdapter, ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	stream := discovery.NewRecordingQueryResultStream()
	adapter.SearchStream(ctx, scope, query, ignoreCache, stream)
	if errs := stream.GetErrors(); len(errs) > 0 {
		return nil, fmt.Errorf("failed to search: %v", errs)
	}
	return stream.GetItems(), nil
}
