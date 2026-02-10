---
title: Autoscaling Group
sidebar_label: autoscaling-auto-scaling-group
---

An AWS Autoscaling Group (ASG) is a logical collection of Amazon EC2 instances that are treated as a single scalable resource. It automatically adjusts the number of running instances to maintain a desired capacity, respond to demand spikes, enforce health‐based replacement, and support rolling updates. Configuration parameters such as minimum, maximum and desired instance counts, scaling policies, health checks and lifecycle hooks are all defined at the group level.  
Further information is available in the official AWS documentation: https://docs.aws.amazon.com/autoscaling/ec2/userguide/AutoScalingGroup.html

**Terrafrom Mappings:**

- `aws_autoscaling_group.arn`

## Supported Methods

- `GET`: Get an Autoscaling Group by name
- `LIST`: List Autoscaling Groups
- `SEARCH`: Search for Autoscaling Groups by ARN

## Possible Links

### [`ec2-launch-template`](/sources/aws/Types/ec2-launch-template)

An ASG normally references a launch template that describes how each EC2 instance should be configured (AMI, instance type, security groups, IAM instance profile, user data, etc.). Therefore the ASG is linked to its associated `ec2-launch-template`.

### [`elbv2-target-group`](/sources/aws/Types/elbv2-target-group)

ASGs can be attached to one or more ALB/NLB target groups so that their member instances are automatically registered and deregistered as they scale. The link shows which `elbv2-target-group`(s) an ASG feeds.

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

The running EC2 instances that currently belong to an ASG are directly related to it. Overmind surfaces this connection so you can see which `ec2-instance` objects are under the control of a specific ASG.

### [`iam-role`](/sources/aws/Types/iam-role)

Autoscaling uses an AWS service-linked role (typically `AWSServiceRoleForAutoScaling`) to perform scaling and health check actions on your behalf. Additionally, the launch template referenced by the ASG may specify an instance profile containing an IAM role for the launched instances. Both relationships are captured via the `iam-role` link.

### [`ec2-placement-group`](/sources/aws/Types/ec2-placement-group)

If the ASG’s launch template specifies a placement group, any instances it launches will be placed accordingly for improved networking performance or spread. The link reveals the `ec2-placement-group` associated with the ASG.
