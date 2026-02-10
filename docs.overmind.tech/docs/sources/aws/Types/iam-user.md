---
title: IAM User
sidebar_label: iam-user
---

An IAM user is a discrete identity within AWS Identity and Access Management that represents a human, service or application which needs to interact with AWS resources. Each user has its own credentials and permissions that determine what actions it can perform in an AWS account. For full details, refer to the AWS documentation: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users.html

**Terrafrom Mappings:**

- `aws_iam_user.arn`
- `aws_iam_user_group_membership.user`

## Supported Methods

- `GET`: Get an IAM user by name
- `LIST`: List all IAM users
- `SEARCH`: Search for IAM users by ARN

## Possible Links

### [`iam-group`](/sources/aws/Types/iam-group)

IAM users can be members of one or more IAM groups, inheriting the group’s managed and inline policies. Overmind therefore links an IAM user to the `iam-group` type whenever the user is listed as a member of that group.
