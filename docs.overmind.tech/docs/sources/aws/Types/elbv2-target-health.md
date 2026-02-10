---
title: ELB Target Health
sidebar_label: elbv2-target-health
---

Elastic Load Balancing (v2) distributes traffic across multiple targets such as EC2 instances, IP addresses, and Lambda functions.
The ELB Target Health resource in Overmind represents the status of a single target as returned by the AWS `DescribeTargetHealth` API.
It shows whether the target is healthy, unhealthy, initialising, or draining, together with any failure reasons, so you can spot issues before a change is deployed.
Official documentation: https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_DescribeTargetHealth.html

## Supported Methods

- `GET`: Get target health by unique ID (`{TargetGroupArn}|{Id}|{AvailabilityZone}|{Port}`)
- ~~`LIST`~~
- `SEARCH`: Search for target health by target group ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

When the target group’s type is `instance`, each registered EC2 instance appears as an ELB target. The target-health record shows whether that particular EC2 instance is currently considered healthy by the load balancer.

### [`lambda-function`](/sources/aws/Types/lambda-function)

For target groups of type `lambda`, the Lambda function itself is the target. The target-health item reports the invocation health of the function as assessed by the load balancer.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

If the target group is of type `ip`, every registered IP address becomes a target. The target-health entry records the health of that IP address, enabling you to see whether traffic will be routed to it.

### [`elbv2-load-balancer`](/sources/aws/Types/elbv2-load-balancer)

The load balancer associated with the target group uses these health results to decide where to send traffic. Linking to the load balancer lets you trace how a target’s health status could affect overall load-balancer behaviour.
