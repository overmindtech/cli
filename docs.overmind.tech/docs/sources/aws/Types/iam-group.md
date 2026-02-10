---
title: IAM Group
sidebar_label: iam-group
---

An IAM (Identity and Access Management) group is a logical collection of IAM users within an AWS account. Permissions—attached to the group via policies—apply to every user who is a member, making it easier to manage access at scale. Because groups do not have their own security credentials, they cannot be used to log in directly; instead, they serve solely as a mechanism for permission inheritance and simplified administration. For full details, refer to the AWS documentation: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_groups.html

**Terrafrom Mappings:**

- `aws_iam_group.arn`

## Supported Methods

- `GET`: Get a group by name
- `LIST`: List all IAM groups
- `SEARCH`: Search for a group by ARN
