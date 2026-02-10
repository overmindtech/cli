---
title: KMS Grant
sidebar_label: kms-grant
---

AWS Key Management Service (KMS) grants are lightweight authorisations that give a specified principal permission to use a particular KMS key for a defined set of operations (such as Encrypt, Decrypt, GenerateDataKey or RetireGrant). Unlike key policies and IAM policies, grants can be created and retired programmatically and have an optional time-to-live, making them ideal for short-lived workloads or delegated access. For a full description see the official AWS documentation: https://docs.aws.amazon.com/kms/latest/developerguide/grants.html

**Terrafrom Mappings:**

- `aws_kms_grant.grant_id`

## Supported Methods

- `GET`: Get a grant by keyID/grantId
- ~~`LIST`~~
- `SEARCH`: Search grants by keyID

## Possible Links

### [`kms-key`](/sources/aws/Types/kms-key)

Every grant is created against exactly one KMS key. The grant specifies which operations are allowed on that key, so the relationship is “KMS key ­— has → grant”.

### [`iam-user`](/sources/aws/Types/iam-user)

An IAM user can appear as the grantee principal or the retiring principal in a grant. If the user is referenced, the link shows which grants give that user access to which keys.

### [`iam-role`](/sources/aws/Types/iam-role)

Similar to IAM users, an IAM role may be listed as the grantee or retiring principal. The link reveals the grants that permit the role to use or retire access to specific KMS keys.
