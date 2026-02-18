---
title: Transit Gateway
sidebar_label: ec2-transit-gateway
---

An AWS Transit Gateway is a network transit hub that you use to interconnect your VPCs and on-premises networks. Each transit gateway has a default route table and can have additional route tables to control routing for attachments (VPCs, VPNs, Direct Connect gateways, peering, Connect).

Official API documentation: [DescribeTransitGateways](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGateways.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway.id`

## Supported Methods

- `GET`: Get a transit gateway by ID
- `LIST`: List all transit gateways
- `SEARCH`: Search transit gateways by ARN
