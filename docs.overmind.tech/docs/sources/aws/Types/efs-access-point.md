---
title: EFS Access Point
sidebar_label: efs-access-point
---

Amazon Elastic File System (EFS) Access Points are application-specific entry points into an EFS file system. Each access point can enforce a unique POSIX user, group and root directory, allowing multiple workloads or tenants to share the same file system while maintaining separation and least-privilege access. Access points are commonly used to simplify permissions when deploying containers, serverless functions or batch jobs that need shared storage.  
Official AWS documentation: https://docs.aws.amazon.com/efs/latest/ug/efs-access-points.html

**Terrafrom Mappings:**

- `aws_efs_access_point.id`

## Supported Methods

- `GET`: Get an access point by ID
- `LIST`: List all access points
- `SEARCH`: Search for an access point by ARN
