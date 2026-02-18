---
title: Transit Gateway Route Table Association
sidebar_label: ec2-transit-gateway-route-table-association
---

An association links a transit gateway attachment (VPC, VPN, Direct Connect gateway, peering, or Connect) to a transit gateway route table. Traffic for that attachment is routed according to the route table.

Official API documentation: [GetTransitGatewayRouteTableAssociations](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_GetTransitGatewayRouteTableAssociations.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway_route_table_association.id`

## Supported Methods

- `GET`: Get by composite ID `TransitGatewayRouteTableId|TransitGatewayAttachmentId`
- `LIST`: List all route table associations (across all route tables in the scope)
- `SEARCH`: Search by `TransitGatewayRouteTableId` to list all associations for that route table (used by the route table’s link to associations)

## Possible Links

### [`ec2-transit-gateway-route-table`](/sources/aws/Types/ec2-transit-gateway-route-table)

The route table that the attachment is associated with.

### [`ec2-transit-gateway-attachment`](/sources/aws/Types/ec2-transit-gateway-attachment)

The transit gateway attachment that is associated with the route table.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

When the attachment resource type is VPC, the linked VPC.

### [`ec2-vpn-connection`](/sources/aws/Types/ec2-vpn-connection)

When the attachment resource type is VPN, the linked VPN connection.

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

When the attachment resource type is Direct Connect gateway, the linked Direct Connect gateway.
