---
title: ECS Service
sidebar_label: ecs-service
---

An Amazon Elastic Container Service (ECS) **service** is the long-running, scalable unit that maintains a specified number of copies of a task definition running on an ECS cluster. The service schedules tasks either on EC2 instances or on Fargate, monitors their health, replaces unhealthy tasks and, when configured, integrates with Elastic Load Balancing and AWS Service Discovery.
For a full description see the AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_services.html

**Terrafrom Mappings:**

- `aws_ecs_service.cluster_name`

## Supported Methods

- `GET`: Get an ECS service by full name (`{clusterName}/{id}`)
- ~~`LIST`~~
- `SEARCH`: Search for ECS services by cluster

## Possible Links

### [`ecs-cluster`](/sources/aws/Types/ecs-cluster)

The service is deployed into exactly one ECS cluster, so each ecs-service will have a **`parent`** relationship to the corresponding `ecs-cluster`.

### [`elbv2-target-group`](/sources/aws/Types/elbv2-target-group)

If the service is configured with a load balancer, it registers its tasks as targets in one or more ELBv2 target groups; Overmind creates a **`uses`** link from the service to every target group referenced in its loadBalancer or serviceConnect configuration.

### [`ecs-task-definition`](/sources/aws/Types/ecs-task-definition)

A service runs a specific revision of a task definition. There is therefore a **`depends_on`** link from the service to the task definition ARN specified in `taskDefinition`.

### [`ecs-capacity-provider`](/sources/aws/Types/ecs-capacity-provider)

When a capacity provider strategy is attached, the service relies on one or more capacity providers for scheduling. Overmind shows a **`uses`** link to each referenced `ecs-capacity-provider`.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

For services that use the `awsvpc` network mode (Fargate or ENI-aware EC2 launch type), the service’s tasks are launched inside specific subnets defined in the service’s network configuration; those subnets are exposed via **`uses`** links.

### [`dns`](/sources/stdlib/Types/dns)

If AWS Cloud Map service discovery is enabled, the ECS service automatically creates DNS records (A, AAAA, or SRV) for its tasks. Overmind surfaces a **`creates`** link to the resultant DNS names.
