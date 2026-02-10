---
title: Target Group
sidebar_label: elbv2-target-group
---

An Amazon Elastic Load Balancing v2 (ELBv2) target group is a logical grouping of targets—such as EC2 instances, IP addresses, Lambda functions or Application Load Balancers—that a load balancer routes traffic to. It contains configuration such as the protocol and port to use, health-check settings, stickiness, deregistration delay and slow-start settings, all within a single VPC. Listeners on an Application Load Balancer (ALB) or Network Load Balancer (NLB) forward requests to one or more target groups based on listener rules.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-target-groups.html

**Terrafrom Mappings:**

- `aws_alb_target_group.arn`
- `aws_lb_target_group.arn`

## Supported Methods

- `GET`: Get a target group by name
- `LIST`: List all target groups
- `SEARCH`: Search for target groups by load balancer ARN or target group ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

A target group is always created within a specific VPC, and all of its registered IP addresses or instance-based targets must reside in that VPC. Therefore the target group is linked to the VPC where its network resources live.

### [`elbv2-load-balancer`](/sources/aws/Types/elbv2-load-balancer)

Load balancers reference target groups in their listener rules. This link shows which load balancers are configured to forward traffic to the target group, or conversely, which target groups a given load balancer depends upon.

### [`elbv2-target-health`](/sources/aws/Types/elbv2-target-health)

Each target group has a corresponding set of target-health descriptions indicating the current health status of every registered target. This link surfaces those health objects so you can see whether the targets in the group are healthy, unhealthy, initialising or unused.
