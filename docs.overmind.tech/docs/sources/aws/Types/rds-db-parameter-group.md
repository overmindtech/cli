---
title: RDS Parameter Group
sidebar_label: rds-db-parameter-group
---

An Amazon RDS DB parameter group is a container for engine configuration values that determine how a database instance or cluster behaves. By attaching a parameter group to one or more RDS resources you override the engine’s built-in defaults with your own settings, allowing you to tune performance, security and operational behaviour. Changes made to the group are propagated to every associated instance; static parameters take effect after the next reboot, while dynamic parameters may apply immediately.  
For a full explanation see the official AWS documentation: [Working with DB parameter groups](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithParamGroups.html).

**Terrafrom Mappings:**

- `aws_db_parameter_group.arn`

## Supported Methods

- `GET`: Get a parameter group by name
- `LIST`: List all parameter groups
- `SEARCH`: Search for a parameter group by ARN
