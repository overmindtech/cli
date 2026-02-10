---
title: Route Table
sidebar_label: ec2-route-table
---

A Route Table in Amazon Virtual Private Cloud (VPC) contains a set of rules, called routes, that determine where network traffic is directed. Each route specifies a destination CIDR block and a target (for example, an Internet Gateway, NAT Gateway, network interface or VPC peering connection). AWS evaluates the routes in the table to decide how packets that leave a subnet are forwarded. A VPC can have multiple route tables, allowing you to implement fine-grained traffic segregation and control.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Route_Tables.html

**Terrafrom Mappings:**

- `aws_route_table.id`
- `aws_route_table_association.route_table_id`
- `aws_default_route_table.default_route_table_id`
- `aws_route.route_table_id`

## Supported Methods

- `GET`: Get a route table by ID
- `LIST`: List all route tables
- `SEARCH`: Search route tables by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

The Route Table is created inside a specific VPC; every table therefore has a one-to-one parent relationship with the VPC in which it resides.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

Subnets are associated with a Route Table. Traffic that originates in a subnet is evaluated against the routes in its associated table. One route table can be linked to many subnets.

### [`ec2-internet-gateway`](/sources/aws/Types/ec2-internet-gateway)

A Route Table may contain a route whose target is an Internet Gateway, enabling outbound IPv4 traffic (and inbound responses) for the subnets that use the table.

### [`ec2-vpc-endpoint`](/sources/aws/Types/ec2-vpc-endpoint)

Interface and Gateway VPC Endpoints can appear as route targets, directing traffic destined for AWS services or private resources through the endpoint.

### [`ec2-egress-only-internet-gateway`](/sources/aws/Types/ec2-egress-only-internet-gateway)

For IPv6 connectivity, a Route Table can include a route to an Egress-only Internet Gateway, allowing outbound-only IPv6 traffic from the associated subnets.

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

An individual EC2 instance can be specified as the route target (using its instance ID) when it is acting as a virtual appliance or host-based router.

### [`ec2-nat-gateway`](/sources/aws/Types/ec2-nat-gateway)

Routes can target a NAT Gateway, providing Internet access for private subnets while keeping the source IP addresses of instances hidden from the public Internet.

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

A specific Elastic Network Interface (ENI) may be used as a route target to forward traffic to appliances such as firewalls or load balancers hosted on that interface.

### [`ec2-vpc-peering-connection`](/sources/aws/Types/ec2-vpc-peering-connection)

When traffic needs to flow between two VPCs, a route whose target is a VPC Peering Connection is added to the Route Table, enabling cross-VPC communication.
