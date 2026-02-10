---
title: VPC Endpoint
sidebar_label: ec2-vpc-endpoint
---

A VPC Endpoint is an elastic network interface or gateway that enables private connectivity between resources inside an Amazon Virtual Private Cloud (VPC) and supported AWS or third-party services, without traversing the public internet. By routing traffic through the AWS network, VPC Endpoints improve security, reduce latency and remove the need for NAT devices, VPNs or Direct Connect links. For full details, see the AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-endpoints.html

**Terrafrom Mappings:**

- `aws_vpc_endpoint.id`

## Supported Methods

- `GET`: Get a VPC Endpoint by ID
- `LIST`: List all VPC Endpoints
- `SEARCH`: Search VPC Endpoints by ARN
