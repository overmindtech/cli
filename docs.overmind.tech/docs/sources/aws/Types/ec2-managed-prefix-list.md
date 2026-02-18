---
title: Managed Prefix List
sidebar_label: ec2-managed-prefix-list
---

A managed prefix list is a set of one or more CIDR blocks that you can reference in security group rules, route table routes, and other network configuration. Transit gateway routes can use a prefix list as the destination instead of a single CIDR.

Official API documentation: [DescribeManagedPrefixLists](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeManagedPrefixLists.html)

**Terraform Mappings:**

- `aws_ec2_managed_prefix_list.id`

## Supported Methods

- `GET`: Get a managed prefix list by ID
- `LIST`: List all managed prefix lists
- `SEARCH`: Search by ARN
