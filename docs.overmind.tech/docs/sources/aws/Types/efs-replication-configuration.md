---
title: EFS Replication Configuration
sidebar_label: efs-replication-configuration
---

An Amazon Elastic File System (EFS) Replication Configuration defines how an EFS file system is asynchronously replicated to another AWS Region or Availability Zone, providing disaster-recovery protection and enhanced data durability. By creating a replication configuration you specify the source file system, the destination Region, and the encryption and retention settings for the replica. Replication occurs automatically and continuously, with recovery point objectives (RPO) typically within minutes, allowing you to fail over quickly if the primary file system becomes unavailable.  
For full details, refer to the AWS documentation: https://docs.aws.amazon.com/efs/latest/ug/efs-replication.html

**Terrafrom Mappings:**

- `aws_efs_replication_configuration.source_file_system_id`

## Supported Methods

- `GET`: Get a replication configuration by file system ID
- `LIST`: List all replication configurations
- `SEARCH`: Search for a replication configuration by ARN
