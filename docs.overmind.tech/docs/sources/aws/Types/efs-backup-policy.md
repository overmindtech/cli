---
title: EFS Backup Policy
sidebar_label: efs-backup-policy
---

An EFS Backup Policy represents the setting on an Amazon Elastic File System (EFS) file system that turns automatic, daily AWS Backup protection on or off. When the policy is enabled, AWS Backup creates incremental backups of the file system and retains them according to the configured backup plan; when it is disabled, the file system is excluded from automated protection. Managing this resource helps ensure that critical data stored in EFS is covered by a consistent backup and retention strategy, reducing the risk of accidental data loss.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/efs/latest/ug/awsbackup.html

**Terrafrom Mappings:**

- `aws_efs_backup_policy.id`

## Supported Methods

- `GET`: Get an Backup Policy by file system ID
- ~~`LIST`~~
- `SEARCH`: Search for an Backup Policy by ARN
