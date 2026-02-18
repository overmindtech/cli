---
title: Transit Gateway Route Table Propagation
sidebar_label: ec2-transit-gateway-route-table-propagation
---

A propagation enables a transit gateway route table to automatically learn routes from an attachment (VPC, VPN, Direct Connect gateway, peering, or Connect). When propagation is enabled, routes from that attachment appear in the route table.

Official API documentation: [GetTransitGatewayRouteTablePropagations](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_GetTransitGatewayRouteTablePropagations.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway_route_table_propagation.id`

## Supported Methods

- `GET`: Get by composite ID `TransitGatewayRouteTableId|TransitGatewayAttachmentId`
- `LIST`: List all route table propagations (across all route tables in the scope)
- `SEARCH`: Search by `TransitGatewayRouteTableId` to list all propagations for that route table (used by the route table’s link to propagations)

## Possible Links

### [`ec2-transit-gateway-route-table`](/sources/aws/Types/ec2-transit-gateway-route-table)

The route table that is propagating routes from the attachment.

### [`ec2-transit-gateway-route-table-association`](/sources/aws/Types/ec2-transit-gateway-route-table-association)

The route table association for the same route table and attachment (same composite ID). Links propagation and association in the graph.

### [`ec2-transit-gateway-attachment`](/sources/aws/Types/ec2-transit-gateway-attachment)

The attachment whose routes are being propagated into the route table.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

When the attachment resource type is VPC, the linked VPC.

### [`ec2-vpn-connection`](/sources/aws/Types/ec2-vpn-connection)

When the attachment resource type is VPN, the linked VPN connection.

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

When the attachment resource type is Direct Connect gateway, the linked Direct Connect gateway.
