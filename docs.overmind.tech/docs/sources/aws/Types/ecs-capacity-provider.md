---
title: Capacity Provider
sidebar_label: ecs-capacity-provider
---

An Amazon ECS capacity provider tells a cluster where its compute capacity comes from and how that capacity should scale. It can point to an Auto Scaling group of EC2 instances or to the serverless Fargate/Fargate Spot capacity pools, and it contains rules that determine when and how instances are launched or terminated to satisfy task demand. Using capacity providers allows platform teams to separate scaling logic from task scheduling and to adopt multiple capacity sources within a single cluster.  
For complete details see the official AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/cluster-capacity-providers.html

**Terrafrom Mappings:**

- `aws_ecs_capacity_provider.arn`

## Supported Methods

- `GET`: Get a capacity provider by its short name or full Amazon Resource Name (ARN).
- `LIST`: List capacity providers.
- `SEARCH`: Search capacity providers by ARN

## Possible Links

### [`autoscaling-auto-scaling-group`](/sources/aws/Types/autoscaling-auto-scaling-group)

A capacity provider that is backed by EC2 instances references exactly one Auto Scaling group. The link lets you trace from the capacity provider to the group that actually supplies instances, making it easy to understand which fleet of instances will scale in response to ECS task demand and to assess risks such as insufficient instance types, mis-configured scaling policies, or conflicting lifecycle hooks.
