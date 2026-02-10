---
title: ELB Instance Health
sidebar_label: elb-instance-health
---

An ELB Instance Health resource represents the current health status of an individual Amazon EC2 instance as reported by an Elastic Load Balancer. The data is returned by the `DescribeInstanceHealth` API call and indicates whether the instance is `InService`, `OutOfService`, or in a transitional state (e.g. `Draining`, `Unknown`). By tracking these objects Overmind can warn you when a deployment will place traffic on unhealthy instances or reduce overall service capacity.
For full details see the AWS documentation: https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/elb-healthchecks.html

## Supported Methods

- `GET`: Get instance health by ID (`{LoadBalancerName}/{InstanceId}`)
- `LIST`: List all instance healths
- ~~`SEARCH`~~

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

Each ELB Instance Health object is intrinsically linked to the EC2 instance whose state it describes. Following this link allows you to inspect configuration details (such as security groups or attached volumes) that may be contributing to an unhealthy status.
