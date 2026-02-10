---
title: Container Instance
sidebar_label: ecs-container-instance
---

A container instance represents an Amazon EC2 host that has been registered to an Amazon ECS cluster and is therefore available for running one or more ECS tasks. Each container instance runs the ECS agent and reports its status, resource availability and running tasks back to the cluster’s control plane. For a detailed explanation of container instances, provisioning requirements, and lifecycle behaviour, see the official AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_instances.html

## Supported Methods

- `GET`: Get a container instance by ID which consists of `{clusterName}/{id}`
- ~~`LIST`~~
- `SEARCH`: Search for container instances by cluster

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

Every container instance is physically an Amazon EC2 instance. Linking to the `ec2-instance` type allows Overmind to surface the underlying compute resource, including its security groups, IAM roles and network configuration, all of which can influence the risk profile of the container instance and any tasks scheduled on it.
