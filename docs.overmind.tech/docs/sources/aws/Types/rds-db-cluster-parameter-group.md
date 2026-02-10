---
title: RDS Cluster Parameter Group
sidebar_label: rds-db-cluster-parameter-group
---

An RDS Cluster Parameter Group is a named collection of engine configuration values that are applied to every instance within an Amazon RDS or Aurora DB cluster. By adjusting the parameters in the group you can fine-tune settings such as memory management, logging, and query optimisation, and have those settings propagated consistently across the cluster. If you do not specify a custom group when you create a cluster, AWS assigns the default engine-specific parameter group. For details, see the AWS documentation: https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/USER_WorkingWithParamGroups.html.

**Terrafrom Mappings:**

- `aws_rds_cluster_parameter_group.arn`

## Supported Methods

- `GET`: Get a parameter group by name
- `LIST`: List all RDS parameter groups
- `SEARCH`: Search for a parameter group by ARN
