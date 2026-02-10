---
title: RDS Option Group
sidebar_label: rds-option-group
---

An Amazon Relational Database Service (RDS) Option Group is a logical container that lets you enable and configure additional features—known as “options”—for an RDS DB instance or cluster. Typical options include Oracle Transparent Data Encryption, SQL Server Audit, MariaDB Audit Plugin and many others that are not activated by default with the engine. By assigning an option group to one or more databases you ensure that each instance inherits the same, centrally-managed configuration, simplifying governance and compliance.  
For complete details see the official AWS documentation: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithOptionGroups.html

**Terrafrom Mappings:**

- `aws_db_option_group.arn`

## Supported Methods

- `GET`: Get an option group by name
- `LIST`: List all RDS option groups
- `SEARCH`: Search for an option group by ARN
