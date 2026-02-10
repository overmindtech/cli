---
title: Task Definition
sidebar_label: ecs-task-definition
---

An Amazon ECS task definition is the blueprint that tells AWS ECS how to run one or more containers. It specifies details such as the container images, CPU and memory requirements, networking mode, logging configuration, IAM roles, and secrets that should be injected into the containers. Each time you register a new version, ECS creates a new immutable revision that can be referenced directly or through the family name.
For full details, see the official AWS documentation: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definitions.html

**Terrafrom Mappings:**

- `aws_ecs_task_definition.family`

## Supported Methods

- `GET`: Get a task definition by revision name (`{family}:{revision}`)
- `LIST`: List all task definitions
- `SEARCH`: Search for task definitions by ARN

## Possible Links

### [`iam-role`](/sources/aws/Types/iam-role)

A task definition can reference an IAM role through `taskRoleArn` and/or `executionRoleArn`. These roles grant the running containers the permissions they need to interact with other AWS services or to pull private images and write logs. Overmind links the task definition to the IAM role resources so you can see the exact permissions that will be in effect at runtime.

### [`ssm-parameter`](/sources/aws/Types/ssm-parameter)

Environment variables or secrets defined in a task definition can be sourced from AWS Systems Manager Parameter Store. Whenever a task definition lists an SSM parameter (e.g., via the `secrets` block), Overmind surfaces a link to the corresponding `ssm-parameter` item, allowing you to trace where sensitive configuration values originate and assess the impact of changes.
