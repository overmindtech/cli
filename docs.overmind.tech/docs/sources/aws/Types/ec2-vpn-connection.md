---
title: VPN Connection
sidebar_label: ec2-vpn-connection
---

An AWS Site-to-Site VPN connection links your on-premises network to your VPC (or to a transit gateway) over an encrypted IPsec tunnel. VPN connections can be attached to a transit gateway for use in a hub-and-spoke topology.

Official API documentation: [DescribeVpnConnections](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpnConnections.html)

**Terraform Mappings:**

- `aws_vpn_connection.id`

## Supported Methods

- `GET`: Get a VPN connection by ID
- `LIST`: List all VPN connections
- `SEARCH`: Search by ARN
