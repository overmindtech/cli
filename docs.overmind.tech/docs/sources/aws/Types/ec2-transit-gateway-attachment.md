---
title: Transit Gateway Attachment
sidebar_label: ec2-transit-gateway-attachment
---

A Transit Gateway attachment connects a resource (VPC, VPN connection, Direct Connect gateway, peering connection, or Connect attachment) to a transit gateway. Attachments are associated with route tables and can have routes propagated to them.

Official API documentation: [DescribeTransitGatewayAttachments](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayAttachments.html)

**Terraform Mappings:**

- `aws_ec2_transit_gateway_vpc_attachment.id` (VPC)
- `aws_ec2_transit_gateway_vpn_attachment.id` (VPN)
- Other attachment types have corresponding Terraform resources.

## Supported Methods

- `GET`: Get a transit gateway attachment by ID
- `LIST`: List all transit gateway attachments
- `SEARCH`: Search by ARN
