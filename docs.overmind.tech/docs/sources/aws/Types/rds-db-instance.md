---
title: RDS Instance
sidebar_label: rds-db-instance
---

Amazon Relational Database Service (RDS) DB instances are the managed compute and storage resources that run your relational database engines in AWS. An instance encapsulates the underlying virtual hardware, disk, network interfaces, and database server software that form a single, addressable database node. Full service description and behaviour are documented in the AWS RDS User Guide: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Overview.DBInstance.html

**Terrafrom Mappings:**

- `aws_db_instance.identifier`
- `aws_db_instance_role_association.db_instance_identifier`

## Supported Methods

- `GET`: Get an instance by ID
- `LIST`: List all instances
- `SEARCH`: Search for instances by ARN

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

Every RDS instance exposes an endpoint such as `mydb.abc123.eu-west-2.rds.amazonaws.com`. Overmind links the instance to the corresponding DNS record so you can trace how applications resolve and reach the database.

### [`route53-hosted-zone`](/sources/aws/Types/route53-hosted-zone)

The automatically-created DNS record for an RDS endpoint lives inside an AWS-managed Route 53 hosted zone, and private zones in your account may contain CNAMEs pointing to it. Overmind surfaces these zones to show where the endpoint is published and overridden.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

In a VPC, an RDS instance is attached to one or more security groups that define its inbound and outbound traffic rules. These links let you audit which networks and EC2 instances are permitted to reach the database.

### [`rds-db-parameter-group`](/sources/aws/Types/rds-db-parameter-group)

A DB parameter group controls engine-level configuration such as `max_connections` or `log_min_duration_statement`. Each instance references exactly one parameter group (or the default), so Overmind links them for configuration drift and compliance checks.

### [`rds-db-subnet-group`](/sources/aws/Types/rds-db-subnet-group)

The subnet group lists the subnets (and therefore the AZs) where the instance may be placed. Linking highlights the network reachability and resiliency zone choices for the database.

### [`rds-db-cluster`](/sources/aws/Types/rds-db-cluster)

For Aurora and other clustered engines, individual DB instances are members of an RDS DB cluster. Overmind links them so you can see the relationship between writer/reader nodes and the cluster-level endpoints.

### [`kms-key`](/sources/aws/Types/kms-key)

When storage encryption is enabled, an RDS instance uses an AWS KMS key to encrypt its underlying EBS volumes and snapshots. The link shows which key protects the data and who can decrypt it.

### [`iam-role`](/sources/aws/Types/iam-role)

Features such as S3 import/export, AWS Lambda integration, and CloudWatch Logs require the database service to assume an IAM service role. Overmind lists these roles so you can review permissions the database can exercise in your account.

### [`iam-instance-profile`](/sources/aws/Types/iam-instance-profile)

RDS Custom instances (and certain on-host integrations) run on dedicated EC2 instances within your account and therefore use an IAM instance profile. If present, Overmind links the profile to reveal any additional permissions granted to the underlying host.
