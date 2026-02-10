---
title: IAM Role
sidebar_label: iam-role
---

An AWS Identity and Access Management (IAM) role is an identity that you can assume to obtain temporary security credentials so that you can make AWS requests. Unlike users, roles do not have long-term credentials; instead, they rely on trust relationships and attached policies to define who can assume the role and what they can do once they have it. IAM roles are typically used for granting permissions to AWS services, cross-account access, or federated users.  
For full details, see the AWS documentation: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html

**Terrafrom Mappings:**

- `aws_iam_role.arn`

## Supported Methods

- `GET`: Get an IAM role by name
- `LIST`: List all IAM roles
- `SEARCH`: Search for IAM roles by ARN

## Possible Links

### [`iam-policy`](/sources/aws/Types/iam-policy)

An IAM role is functionally useless without one or more IAM policies attached to it. Overmind links an `iam-role` to the `iam-policy` resources that 1) are attached as inline or managed policies granting permissions, and 2) define the trust relationship (the role’s assume-role policy). This allows you to trace which permissions the role grants and who or what is allowed to assume it.
