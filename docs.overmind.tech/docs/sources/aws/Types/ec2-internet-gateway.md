---
title: Internet Gateway
sidebar_label: ec2-internet-gateway
---

An Internet Gateway is a highly-available, horizontally-scaled component that provides a Virtual Private Cloud (VPC) with a route to the public Internet. When attached to a VPC and referenced in the route table, it enables resources with public IP addresses—such as EC2 instances, NAT gateways or load balancers—to send and receive traffic to and from the wider Internet. Because it is a managed AWS service, it does not introduce any single point of failure and requires no administration beyond attachment and routing.  
For the official AWS documentation, see https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html.

**Terrafrom Mappings:**

- `aws_internet_gateway.id`

## Supported Methods

- `GET`: Get an internet gateway by ID
- `LIST`: List all internet gateways
- `SEARCH`: Search internet gateways by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

An Internet Gateway must be attached to exactly one VPC; this link represents that one-to-one relationship. Through it, Overmind can surface configuration drift (for example, if the gateway is detached) and highlight risks such as missing or overly permissive route-table entries that would expose private resources to the Internet.
