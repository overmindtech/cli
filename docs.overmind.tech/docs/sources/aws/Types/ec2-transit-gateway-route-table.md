---
title: Transit Gateway Route Table
sidebar_label: ec2-transit-gateway-route-table
---

A Transit Gateway Route Table determines how traffic is routed for attachments (VPCs, VPNs, Direct Connect gateways, peering connections, or Connect attachments) that are associated with it. Each transit gateway has a default route table; you can create additional route tables to control which attachments can reach which routes.

Official API documentation: [DescribeTransitGatewayRouteTables](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway_route_table.id`

## Supported Methods

- `GET`: Get a transit gateway route table by ID
- `LIST`: List all transit gateway route tables
- `SEARCH`: Search transit gateway route tables by ARN

## Possible Links

### [`ec2-transit-gateway`](/sources/aws/Types/ec2-transit-gateway)

Each transit gateway route table belongs to a single transit gateway. The route table controls routing for attachments that are associated with it.

### [`ec2-transit-gateway-route-table-association`](/sources/aws/Types/ec2-transit-gateway-route-table-association)

Associations for this route table (Search by route table ID). Each association links an attachment to this route table.

### [`ec2-transit-gateway-route-table-propagation`](/sources/aws/Types/ec2-transit-gateway-route-table-propagation)

Propagations for this route table (Search by route table ID). Each propagation enables the route table to learn routes from an attachment.

### [`ec2-transit-gateway-route`](/sources/aws/Types/ec2-transit-gateway-route)

Routes in this route table (Search by route table ID).
