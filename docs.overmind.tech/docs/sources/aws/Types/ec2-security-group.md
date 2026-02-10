---
title: Security Group
sidebar_label: ec2-security-group
---

An Amazon EC2 Security Group acts as a virtual firewall that regulates inbound and outbound traffic for resources such as EC2 instances, load balancers, and network interfaces within a Virtual Private Cloud (VPC). Rules are stateful, meaning that return traffic is automatically allowed, and can be specified by protocol, port range, and source or destination (CIDR block, prefix list, or another security group). For further details, refer to the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/VPC_SecurityGroups.html

**Terrafrom Mappings:**

- `aws_security_group.id`
- `aws_security_group_rule.security_group_id`

## Supported Methods

- `GET`: Get a security group by ID
- `LIST`: List all security groups
- `SEARCH`: Search for security groups by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Each security group is created within a single VPC, inherits its CIDR boundaries, and can only be attached to resources that also reside in that VPC.
