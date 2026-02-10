---
title: ECS Task
sidebar_label: ecs-task
---

An ECS task is the fundamental unit of work that runs on Amazon Elastic Container Service (ECS). It represents one instantiation of a task definition: a group of one or more Docker containers that are deployed together on the same host. A task lives within an ECS cluster and may run on EC2 instances or on AWS Fargate. The task record captures runtime information such as status, start/stop times, allocated network interfaces and resource utilisation.  
For full details, see the AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_tasks.html

## Supported Methods

- `GET`: Get an ECS task by ID
- ~~`LIST`~~
- `SEARCH`: Search for ECS tasks by cluster

## Possible Links

### [`ecs-cluster`](/sources/aws/Types/ecs-cluster)

The task is launched inside exactly one ECS cluster, so Overmind links each task back to the cluster that owns it.

### [`ecs-container-instance`](/sources/aws/Types/ecs-container-instance)

For tasks that use the EC2 launch type, the task runs on a specific ECS container instance (an EC2 host registered with the cluster). Overmind links the task to the container instance on which it is currently placed.

### [`ecs-task-definition`](/sources/aws/Types/ecs-task-definition)

Every task is an instantiation of a task definition. Overmind records this relationship so you can trace configuration changes in the task definition that may affect a running task.

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

When a task uses the `awsvpc` network mode (or is a Fargate task), AWS allocates one or more elastic network interfaces (ENIs) to the task. These ENIs are linked so you can observe associated security groups, subnets and IP addresses.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Each ENI attached to the task is assigned private (and optionally public) IP addresses. Overmind surfaces these IP resources, allowing you to see which IPs are in use by a given task and how they propagate through your network topology.
