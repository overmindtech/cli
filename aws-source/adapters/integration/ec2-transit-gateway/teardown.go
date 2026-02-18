package ec2transitgateway

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

// integrationTestTagFilters returns filters to discover resources created by this suite.
func integrationTestTagFilters() []types.Filter {
	return []types.Filter{
		{Name: ptr("tag:" + integration.TagTestKey), Values: []string{integration.TagTestValue}},
		{Name: ptr("tag:" + integration.TagTestIDKey), Values: []string{integrationTestName}},
	}
}

// getIntegrationTestTransitGatewayID returns the transit gateway ID for the integration-test
// resources. If Setup ran in this process, it uses the package-level ID; otherwise it
// discovers the TGW by tag so tests work when run after a separate Setup (e.g. a day later).
// Returns an error if no tagged TGW is found (e.g. after Teardown).
func getIntegrationTestTransitGatewayID(ctx context.Context, client *ec2.Client) (string, error) {
	if createdTransitGatewayID != "" {
		return createdTransitGatewayID, nil
	}
	tgwOut, err := client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{
		Filters: integrationTestTagFilters(),
	})
	if err != nil {
		return "", err
	}
	for _, tgw := range tgwOut.TransitGateways {
		if tgw.TransitGatewayId != nil && tgw.State != types.TransitGatewayStateDeleted && tgw.State != types.TransitGatewayStateDeleting {
			return *tgw.TransitGatewayId, nil
		}
	}
	return "", errors.New("no transit gateway found with integration-test tag (run Setup first or ensure Teardown has not deleted resources)")
}

func Teardown(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	client, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	if err := teardown(ctx, logger, client); err != nil {
		t.Fatalf("Teardown failed: %v", err)
	}
}

func teardown(ctx context.Context, logger *slog.Logger, client *ec2.Client) error {
	tagFilters := integrationTestTagFilters()

	// 1. Discover transit gateways by tag.
	tgwOut, err := client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{
		Filters: tagFilters,
	})
	if err != nil {
		return err
	}
	if len(tgwOut.TransitGateways) == 0 {
		logger.InfoContext(ctx, "No transit gateways found with integration-test tag")
		clearPackageState()
		return nil
	}

	// 2. For each TGW: delete static route, then VPC attachments, and wait for attachments to be deleted.
	for _, tgw := range tgwOut.TransitGateways {
		if tgw.TransitGatewayId == nil || tgw.State == types.TransitGatewayStateDeleted || tgw.State == types.TransitGatewayStateDeleting {
			continue
		}
		tgwID := *tgw.TransitGatewayId

		// Resolve default route table and delete our static route.
		rtOut, err := client.DescribeTransitGatewayRouteTables(ctx, &ec2.DescribeTransitGatewayRouteTablesInput{
			Filters: []types.Filter{{Name: ptr("transit-gateway-id"), Values: []string{tgwID}}},
		})
		if err != nil {
			return err
		}
		var defaultRouteTableID string
		for i := range rtOut.TransitGatewayRouteTables {
			rt := &rtOut.TransitGatewayRouteTables[i]
			if rt.TransitGatewayRouteTableId != nil && rt.DefaultAssociationRouteTable != nil && *rt.DefaultAssociationRouteTable {
				defaultRouteTableID = *rt.TransitGatewayRouteTableId
				break
			}
		}
		if defaultRouteTableID != "" {
			_, _ = client.DeleteTransitGatewayRoute(ctx, &ec2.DeleteTransitGatewayRouteInput{
				TransitGatewayRouteTableId: &defaultRouteTableID,
				DestinationCidrBlock:       &createdRouteDestination,
			})
		}

		// List VPC attachments for this TGW and delete each.
		attachOut, err := client.DescribeTransitGatewayVpcAttachments(ctx, &ec2.DescribeTransitGatewayVpcAttachmentsInput{
			Filters: []types.Filter{{Name: ptr("transit-gateway-id"), Values: []string{tgwID}}},
		})
		if err != nil {
			return err
		}
		for _, att := range attachOut.TransitGatewayVpcAttachments {
			if att.TransitGatewayAttachmentId == nil || att.State == types.TransitGatewayAttachmentStateDeleted || att.State == types.TransitGatewayAttachmentStateDeleting {
				continue
			}
			attID := *att.TransitGatewayAttachmentId
			_, _ = client.DeleteTransitGatewayVpcAttachment(ctx, &ec2.DeleteTransitGatewayVpcAttachmentInput{
				TransitGatewayAttachmentId: &attID,
			})
			logger.InfoContext(ctx, "Deleted TGW VPC attachment, waiting for deleted", "id", attID)
			deadline := time.Now().Add(5 * time.Minute)
			for time.Now().Before(deadline) {
				desc, err := client.DescribeTransitGatewayVpcAttachments(ctx, &ec2.DescribeTransitGatewayVpcAttachmentsInput{
					TransitGatewayAttachmentIds: []string{attID},
				})
				if err != nil || len(desc.TransitGatewayVpcAttachments) == 0 {
					break
				}
				if desc.TransitGatewayVpcAttachments[0].State == types.TransitGatewayAttachmentStateDeleted {
					break
				}
				time.Sleep(10 * time.Second)
			}
		}
	}

	// 3. Delete subnets by tag.
	subOut, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: tagFilters})
	if err != nil {
		return err
	}
	for _, sub := range subOut.Subnets {
		if sub.SubnetId != nil {
			_, _ = client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: sub.SubnetId})
		}
	}

	// 4. Delete VPCs by tag.
	vpcOut, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{Filters: tagFilters})
	if err != nil {
		return err
	}
	for _, vpc := range vpcOut.Vpcs {
		if vpc.VpcId != nil {
			_, _ = client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: vpc.VpcId})
		}
	}

	// 5. Delete transit gateways.
	for _, tgw := range tgwOut.TransitGateways {
		if tgw.TransitGatewayId == nil || tgw.State == types.TransitGatewayStateDeleted || tgw.State == types.TransitGatewayStateDeleting {
			continue
		}
		tgwID := *tgw.TransitGatewayId
		_, err := client.DeleteTransitGateway(ctx, &ec2.DeleteTransitGatewayInput{TransitGatewayId: &tgwID})
		if err != nil {
			return err
		}
		logger.InfoContext(ctx, "Deleted transit gateway", "id", tgwID)
	}

	clearPackageState()
	return nil
}

func clearPackageState() {
	createdTransitGatewayID = ""
	createdRouteTableID = ""
	createdVpcID = ""
	createdSubnetID = ""
	createdAttachmentID = ""
}
