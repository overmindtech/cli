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

// integrationTestName is the fixed tag value and name for all resources created by
// this suite. Teardown discovers and deletes resources by tag test-id=<value>, so
// it can be run alone to clean stale resources from previous runs.
const integrationTestName = "integration-test"

// Package-level state set by Setup and used by tests and Teardown.
var (
	createdTransitGatewayID  string
	createdRouteTableID     string
	createdVpcID            string
	createdSubnetID         string
	createdAttachmentID     string
	createdRouteDestination = "10.88.0.0/16" // static route we create (distinct from VPC CIDR)
)

func Setup(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	client, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	if err := setup(ctx, logger, client); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
}

func setup(ctx context.Context, logger *slog.Logger, client *ec2.Client) error {
	out, err := client.CreateTransitGateway(ctx, &ec2.CreateTransitGatewayInput{
		Description: ptr("Overmind " + integrationTestName),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeTransitGateway,
				Tags: []types.Tag{
					{Key: ptr(integration.TagTestKey), Value: ptr(integration.TagTestValue)},
					{Key: ptr(integration.TagTestIDKey), Value: ptr(integrationTestName)},
					{Key: ptr("Name"), Value: ptr(integrationTestName)},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	if out.TransitGateway == nil || out.TransitGateway.TransitGatewayId == nil {
		return errors.New("CreateTransitGateway returned nil transit gateway or id")
	}

	tgwID := *out.TransitGateway.TransitGatewayId
	createdTransitGatewayID = tgwID
	logger.InfoContext(ctx, "Created transit gateway, waiting for available", "id", tgwID)

	// Wait for transit gateway to become available (creates default route table).
	const waitTimeout = 5 * time.Minute
	deadline := time.Now().Add(waitTimeout)
	tgwAvailable := false
	for time.Now().Before(deadline) {
		desc, err := client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{
			TransitGatewayIds: []string{tgwID},
		})
		if err != nil {
			return err
		}
		if len(desc.TransitGateways) == 0 {
			time.Sleep(10 * time.Second)
			continue
		}
		state := desc.TransitGateways[0].State
		if state == types.TransitGatewayStateAvailable {
			tgwAvailable = true
			break
		}
		if state == types.TransitGatewayStateDeleted || state == types.TransitGatewayStateDeleting {
			return errors.New("transit gateway entered deleted/deleting state")
		}
		time.Sleep(10 * time.Second)
	}
	if !tgwAvailable {
		return errors.New("timeout waiting for transit gateway to become available")
	}

	// Resolve default route table for this TGW (needed for attachment and static route).
	rtOut, err := client.DescribeTransitGatewayRouteTables(ctx, &ec2.DescribeTransitGatewayRouteTablesInput{
		Filters: []types.Filter{
			{Name: ptr("transit-gateway-id"), Values: []string{tgwID}},
		},
	})
	if err != nil {
		return err
	}
	for i := range rtOut.TransitGatewayRouteTables {
		rt := &rtOut.TransitGatewayRouteTables[i]
		if rt.TransitGatewayRouteTableId != nil && rt.DefaultAssociationRouteTable != nil && *rt.DefaultAssociationRouteTable {
			createdRouteTableID = *rt.TransitGatewayRouteTableId
			break
		}
	}
	if createdRouteTableID == "" {
		return errors.New("could not find default route table for transit gateway")
	}

	// Create VPC and subnet so we can create a VPC attachment (association + propagation + route target).
	vpcOut, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: ptr("10.99.0.0/16"),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{Key: ptr(integration.TagTestKey), Value: ptr(integration.TagTestValue)},
					{Key: ptr(integration.TagTestIDKey), Value: ptr(integrationTestName)},
					{Key: ptr("Name"), Value: ptr(integrationTestName)},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	if vpcOut.Vpc == nil || vpcOut.Vpc.VpcId == nil {
		return errors.New("CreateVpc returned nil vpc or id")
	}
	createdVpcID = *vpcOut.Vpc.VpcId
	logger.InfoContext(ctx, "Created VPC for TGW attachment", "id", createdVpcID)

	// Pick one AZ for the subnet.
	azOut, err := client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []types.Filter{
			{Name: ptr("state"), Values: []string{"available"}},
		},
	})
	if err != nil || len(azOut.AvailabilityZones) == 0 {
		return errors.New("could not describe availability zones")
	}
	az := azOut.AvailabilityZones[0].ZoneName

	subOut, err := client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:            &createdVpcID,
		CidrBlock:        ptr("10.99.1.0/24"),
		AvailabilityZone: az,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{Key: ptr(integration.TagTestKey), Value: ptr(integration.TagTestValue)},
					{Key: ptr(integration.TagTestIDKey), Value: ptr(integrationTestName)},
					{Key: ptr("Name"), Value: ptr(integrationTestName)},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	if subOut.Subnet == nil || subOut.Subnet.SubnetId == nil {
		return errors.New("CreateSubnet returned nil subnet or id")
	}
	createdSubnetID = *subOut.Subnet.SubnetId
	logger.InfoContext(ctx, "Created subnet for TGW attachment", "id", createdSubnetID)

	attachOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &ec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: &tgwID,
		VpcId:            &createdVpcID,
		SubnetIds:        []string{createdSubnetID},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeTransitGatewayAttachment,
				Tags: []types.Tag{
					{Key: ptr(integration.TagTestKey), Value: ptr(integration.TagTestValue)},
					{Key: ptr(integration.TagTestIDKey), Value: ptr(integrationTestName)},
					{Key: ptr("Name"), Value: ptr(integrationTestName)},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	if attachOut.TransitGatewayVpcAttachment == nil || attachOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		return errors.New("CreateTransitGatewayVpcAttachment returned nil attachment or id")
	}
	createdAttachmentID = *attachOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId
	logger.InfoContext(ctx, "Created TGW VPC attachment, waiting for available", "id", createdAttachmentID)

	// Wait for attachment to become available so we can create a route and so associations/propagations appear.
	attachDeadline := time.Now().Add(waitTimeout)
	attachmentAvailable := false
	for time.Now().Before(attachDeadline) {
		desc, err := client.DescribeTransitGatewayVpcAttachments(ctx, &ec2.DescribeTransitGatewayVpcAttachmentsInput{
			TransitGatewayAttachmentIds: []string{createdAttachmentID},
		})
		if err != nil {
			return err
		}
		if len(desc.TransitGatewayVpcAttachments) == 0 {
			time.Sleep(10 * time.Second)
			continue
		}
		state := desc.TransitGatewayVpcAttachments[0].State
		if state == types.TransitGatewayAttachmentStateAvailable {
			attachmentAvailable = true
			break
		}
		if state == types.TransitGatewayAttachmentStateDeleted || state == types.TransitGatewayAttachmentStateDeleting {
			return errors.New("transit gateway VPC attachment entered deleted/deleting state")
		}
		time.Sleep(10 * time.Second)
	}
	if !attachmentAvailable {
		return errors.New("timeout waiting for transit gateway VPC attachment to become available")
	}

	// Add a static route so the route adapter returns at least one item.
	_, err = client.CreateTransitGatewayRoute(ctx, &ec2.CreateTransitGatewayRouteInput{
		TransitGatewayRouteTableId: &createdRouteTableID,
		DestinationCidrBlock:       &createdRouteDestination,
		TransitGatewayAttachmentId: &createdAttachmentID,
	})
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "Created static TGW route", "destination", createdRouteDestination)

	return nil
}

func ptr(s string) *string { return &s }
