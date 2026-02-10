---
title: Security Group Rule
sidebar_label: ec2-security-group-rule
---

A Security Group Rule represents a single ingress or egress rule that belongs to an Amazon EC2 Security Group. Each rule specifies the protocol, port range, source or destination (IP range, prefix list, security group or prefix), and (optionally) a description that determines whether specific network traffic is allowed to reach, or leave, the resources associated with the parent security group. By analysing these rules, Overmind can surface unintended exposure, overly-permissive access, or conflicts before the configuration is deployed.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/security-group-rules.html

**Terrafrom Mappings:**

- `aws_security_group_rule.security_group_rule_id`
- `aws_vpc_security_group_ingress_rule.security_group_rule_id`
- `aws_vpc_security_group_egress_rule.security_group_rule_id`

## Supported Methods

- `GET`: Get a security group rule by ID
- `LIST`: List all security group rules
- `SEARCH`: Search security group rules by ARN

## Possible Links

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Every Security Group Rule belongs to exactly one Security Group; Overmind links the rule back to its parent security group so you can trace how an individual rule contributes to the overall ingress or egress policy applied to your instances and other resources.
