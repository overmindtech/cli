---
title: VPC Peering Connection
sidebar_label: ec2-vpc-peering-connection
---

A VPC Peering Connection enables you to route traffic privately between two Virtual Private Clouds (VPCs) without traversing the public internet. Peering can be established between VPCs in the same AWS account or across different AWS accounts, and—subject to region support—across regions. It is commonly used for micro-service communication, shared services networks, or multi-account architectures where low-latency, high-bandwidth connectivity with AWS-managed security controls is required.  
For full details, refer to the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html

**Terrafrom Mappings:**

- `aws_vpc_peering_connection.id`
- `aws_vpc_peering_connection_accepter.id`
- `aws_vpc_peering_connection_options.vpc_peering_connection_id`

## Supported Methods

- `GET`: Get a VPC Peering Connection by ID
- `LIST`: List all VPC Peering Connections
- `SEARCH`: Search for VPC Peering Connections by their ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Each VPC Peering Connection has exactly two endpoints—a requester VPC and an accepter VPC. Linking to the `ec2-vpc` resource allows Overmind to show which VPCs are joined by a given peering connection and, conversely, which peering connections a particular VPC participates in.
