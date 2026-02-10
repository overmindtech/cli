---
title: ECS Cluster
sidebar_label: ecs-cluster
---

An Amazon ECS (Elastic Container Service) cluster is a logical grouping of tasks or services. It acts as the fundamental boundary for scheduling, networking and capacity management in ECS: every task or service is launched into exactly one cluster, and the cluster manages the resources on which containers run.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/clusters.html

**Terrafrom Mappings:**

- `aws_ecs_cluster.arn`

## Supported Methods

- `GET`: Get a cluster by name
- `LIST`: List all clusters
- `SEARCH`: Search for a cluster by ARN

## Possible Links

### [`ecs-container-instance`](/sources/aws/Types/ecs-container-instance)

An ECS cluster is composed of zero or more container instances (EC2 hosts or AWS Fargate-managed capacity). Each `ecs-container-instance` record represents a specific compute resource that has registered itself to the cluster and is available for running tasks.

### [`ecs-service`](/sources/aws/Types/ecs-service)

Services define long-running workloads that are maintained by ECS within the cluster. Every `ecs-service` is created inside a particular cluster and relies on the cluster’s scheduler to place and maintain tasks according to the service definition.

### [`ecs-task`](/sources/aws/Types/ecs-task)

Tasks are the running instantiations of container definitions. When a task is started, it is launched into a specific cluster; therefore every `ecs-task` is linked back to the cluster that provided the capacity and networking for it.

### [`ecs-capacity-provider`](/sources/aws/Types/ecs-capacity-provider)

Capacity providers control how ECS acquires compute capacity for a cluster (e.g. Fargate, Auto Scaling groups). A cluster may have one or more `ecs-capacity-provider` resources associated with it, and those associations determine how tasks and services within the cluster obtain the underlying compute resources they require.
