---
title: Transit Gateway Route
sidebar_label: ec2-transit-gateway-route
---

A route in a transit gateway route table. Each route has a destination (CIDR or prefix list) and a target (attachment or resource). Routes can be static or propagated from attachments.

Official API documentation: [SearchTransitGatewayRoutes](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SearchTransitGatewayRoutes.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway_route.id`

## Supported Methods

- `GET`: Get by composite ID `TransitGatewayRouteTableId|Destination`, where Destination is a CIDR (e.g. `10.0.0.0/16`) or prefix list (e.g. `pl:PrefixListId`)
- `LIST`: List all transit gateway routes (across all route tables in the scope)
- `SEARCH`: Search by `TransitGatewayRouteTableId` to list all routes in that route table (used by the route table’s link to routes)

## Possible Links

### [`ec2-transit-gateway-route-table`](/sources/aws/Types/ec2-transit-gateway-route-table)

The route table that contains this route.

### [`ec2-transit-gateway-route-table-association`](/sources/aws/Types/ec2-transit-gateway-route-table-association)

For each attachment that this route targets, the corresponding route table association (same route table and attachment). Links routes and associations in the graph.

### [`ec2-transit-gateway-attachment`](/sources/aws/Types/ec2-transit-gateway-attachment)

Each attachment that this route targets (from the route’s `TransitGatewayAttachments`).

### [`ec2-transit-gateway-route-table-announcement`](/sources/aws/Types/ec2-transit-gateway-route-table-announcement)

When the route originates from a route table announcement, the linked transit gateway route table announcement.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

When a route attachment’s resource type is VPC, the linked VPC.

### [`ec2-vpn-connection`](/sources/aws/Types/ec2-vpn-connection)

When a route attachment’s resource type is VPN, the linked VPN connection.

### [`ec2-managed-prefix-list`](/sources/aws/Types/ec2-managed-prefix-list)

When the route destination is a prefix list (instead of a CIDR), the managed prefix list.

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

When a route attachment’s resource type is Direct Connect gateway, the linked Direct Connect gateway.
