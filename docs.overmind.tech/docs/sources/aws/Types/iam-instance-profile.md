---
title: IAM Instance Profile
sidebar_label: iam-instance-profile
---

An IAM Instance Profile is a logical container for an IAM role that you can attach to an Amazon EC2 instance when it is launched. The profile passes the role’s credentials to the instance so that applications running on the instance can securely call other AWS services without embedding long-lived access keys in the code or configuration. For full details see the AWS documentation: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html

**Terrafrom Mappings:**

- `aws_iam_instance_profile.arn`

## Supported Methods

- `GET`: Get an IAM instance profile by name
- `LIST`: List all IAM instance profiles
- `SEARCH`: Search IAM instance profiles by ARN

## Possible Links

### [`iam-role`](/sources/aws/Types/iam-role)

Every instance profile contains exactly one IAM role (though a role can exist without an instance profile). Overmind links the profile to the role it encapsulates so that you can see which permissions will be passed to the EC2 instance.

### [`iam-policy`](/sources/aws/Types/iam-policy)

Policies are not attached directly to the instance profile but to the role inside it. Overmind surfaces these indirect relationships so that you can trace what policies – and therefore permissions – will ultimately be available on the instance through the profile.
