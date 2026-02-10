---
title: EC2 Subnet
sidebar_label: ec2-subnet
---

An EC2 subnet is a logically isolated section of an Amazon Virtual Private Cloud that lets you group resources together and control how traffic flows to and from them. Each subnet resides in a single Availability Zone, inherits the VPC’s CIDR range, and can be configured as public or private depending on whether its routing table points traffic to an Internet Gateway or not. Subnets form the basic building blocks for networking in AWS, determining IP addressing, network reachability, and security-group/network-ACL boundaries.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/configure-subnets.html

**Terrafrom Mappings:**

- `aws_route_table_association.subnet_id`
- `aws_subnet.id`

## Supported Methods

- `GET`: Get a subnet by ID
- `LIST`: List all subnets
- `SEARCH`: Search for subnets by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Every subnet must belong to exactly one VPC. This relationship allows Overmind to trace how traffic is routed from the subnet through VPC-level components such as Internet Gateways, NAT Gateways, route tables, and network ACLs.
