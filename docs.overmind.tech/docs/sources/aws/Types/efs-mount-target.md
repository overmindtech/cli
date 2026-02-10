---
title: EFS Mount Target
sidebar_label: efs-mount-target
---

An EFS Mount Target is a network endpoint that resides in a specific subnet inside your VPC and exposes an Amazon Elastic File System (EFS) file system to compute resources such as EC2 instances, ECS tasks, Lambda functions and other AWS services. By creating one mount target in each Availability Zone where the file system will be accessed, you ensure low-latency, highly available access to shared file storage. Each mount target can be associated with one or more security groups, allowing fine-grained control over which clients can connect to the file system.  
For further details, refer to the official AWS documentation: https://docs.aws.amazon.com/efs/latest/ug/efs-mount-targets.html

**Terrafrom Mappings:**

- `aws_efs_mount_target.id`

## Supported Methods

- `GET`: Get an mount target by ID
- ~~`LIST`~~
- `SEARCH`: Search for mount targets by file system ID
