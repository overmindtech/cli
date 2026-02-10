---
title: RDS Subnet Group
sidebar_label: rds-db-subnet-group
---

An RDS DB subnet group is a named collection of one or more subnets that belong to a single Amazon VPC. When you create an Amazon RDS DB instance in a VPC, the subnet group tells RDS which subnets, and therefore which Availability Zones, it may use to provision and maintain the instance. Subnet groups are essential for ensuring high availability and proper network isolation of database workloads.  
For full details, see the AWS documentation: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_VPC.Subnets.html

**Terrafrom Mappings:**

- `aws_db_subnet_group.arn`

## Supported Methods

- `GET`: Get a subnet group by name
- `LIST`: List all subnet groups
- `SEARCH`: Search for subnet groups by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

The DB subnet group is created within exactly one VPC; its subnets must all belong to this VPC, so the group inherits the VPC’s routing and network-security boundaries.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

A DB subnet group is a container for multiple EC2 subnets, typically spanning at least two Availability Zones. Each listed subnet in the group contributes one possible placement zone for RDS DB instances.
