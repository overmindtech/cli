---
title: Network Interface Permission
sidebar_label: ec2-network-interface-permission
---

An EC2 **Network Interface Permission** represents the right of an AWS principal (usually another AWS account) to attach a specific Elastic Network Interface (ENI) to an instance in that principal’s account. By creating or revoking these permissions you can share network interfaces across accounts in a controlled manner without transferring ownership.  
Further information can be found in the AWS official documentation: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_NetworkInterfacePermission.html

## Supported Methods

- `GET`: Get a network interface permission by ID
- `LIST`: List all network interface permissions
- `SEARCH`: Search network interface permissions by ARN

## Possible Links

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

A network interface permission is always associated with a single network interface; the linked `ec2-network-interface` item is the ENI to which this permission applies.
