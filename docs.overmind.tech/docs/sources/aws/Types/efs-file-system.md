---
title: EFS File System
sidebar_label: efs-file-system
---

Amazon Elastic File System (EFS) provides a scalable, elastic and fully-managed Network File System (NFS) that can be mounted concurrently by multiple AWS compute services, including EC2, Lambda and containers. It automatically grows and shrinks as you add or remove data, removing the need to provision storage up front, and offers high availability across multiple Availability Zones. For a full overview, refer to the official AWS documentation: https://docs.aws.amazon.com/efs/latest/ug/whatisefs.html

**Terrafrom Mappings:**

- `aws_efs_file_system.id`

## Supported Methods

- `GET`: Get a file system by ID
- `LIST`: List file systems
- `SEARCH`: Search file systems by ARN
