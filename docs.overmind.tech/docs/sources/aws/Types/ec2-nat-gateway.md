---
title: NAT Gateway
sidebar_label: ec2-nat-gateway
---

A NAT Gateway is an AWS managed network appliance that enables instances in a private subnet to initiate outbound IPv4 (and, in the case of an **NAT Gateway (v2)**, IPv6) traffic to the internet or other AWS services, while preventing unsolicited inbound connections from the public internet. It provides higher bandwidth and easier management compared to NAT instances, and is designed to be highly available within an Availability Zone.  
For a full description of its features and limitations, see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-nat-gateway.html

**Terrafrom Mappings:**

- `aws_nat_gateway.id`

## Supported Methods

- `GET`: Get a NAT Gateway by ID
- `LIST`: List all NAT gateways
- `SEARCH`: Search for NAT gateways by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

The NAT Gateway is always created inside a specific VPC; this link lets you trace which virtual network the gateway belongs to.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

A NAT Gateway is placed in exactly one subnet. This link shows the subnet that hosts the gateway’s elastic network interface.

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

Each NAT Gateway is automatically assigned an elastic network interface (ENI). Following this link reveals the ENI that represents the gateway inside the subnet.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

When you create a NAT Gateway you must allocate at least one Elastic IP address. This link connects the gateway to the public IP(s) it advertises to the internet.
