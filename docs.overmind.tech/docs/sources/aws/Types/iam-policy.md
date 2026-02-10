---
title: IAM Policy
sidebar_label: iam-policy
---

An IAM policy is a standalone document that defines a set of permissions which determine whether a principal (user, group, or role) is allowed or denied the ability to call specific AWS APIs. Policies are expressed in JSON, may be created and managed by customers or AWS, and are attached to identities or resources to enforce least-privilege access. See the official AWS documentation for full details: https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html

**Terrafrom Mappings:**

- `aws_iam_policy.arn`
- `aws_iam_user_policy_attachment.policy_arn`

## Supported Methods

## Supported Methods

- `GET`: Get a policy by ARN or path. `{path}` is extracted from the ARN path component.
- `LIST`: List all policies
- `SEARCH`: Search for IAM policies by ARN

## Possible Links

### [`iam-group`](/sources/aws/Types/iam-group)

An IAM policy can be attached to an IAM group to grant all members of the group the permissions described in the policy. Overmind therefore links a policy to any groups to which it is attached.

### [`iam-user`](/sources/aws/Types/iam-user)

An IAM policy may be directly attached to an individual IAM user, granting that user the specified permissions. Overmind surfaces this relationship so you can see every user that inherits rights from the policy.

### [`iam-role`](/sources/aws/Types/iam-role)

IAM roles often receive permissions through attached policies. Overmind links a policy to any roles that reference it, allowing you to trace which compute workloads or federated identities can exercise the policy’s privileges.
