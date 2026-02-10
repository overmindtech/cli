---
title: VPC
sidebar_label: ec2-vpc
---

An Amazon Virtual Private Cloud (VPC) is a logically isolated section of AWS in which you can launch and manage AWS resources within a virtual network that you define. Within a VPC you control IP address ranges, subnets, route tables, network gateways, security groups, and network ACLs, allowing you to shape how traffic flows to and from your workloads while keeping them isolated from, or connected to, the public Internet and other VPCs as required. For a full overview, see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/what-is-amazon-vpc.html.

**Terrafrom Mappings:**

- `aws_vpc.id`

## Supported Methods

- `GET`: Get a VPC by ID
- `LIST`: List all VPCs
- ~~`SEARCH`~~
