---
title: KMS Key Policy
sidebar_label: kms-key-policy
---

AWS Key Management Service (KMS) key policies are the primary access-control mechanism for customer-managed KMS keys. A key policy is a JSON document attached directly to a KMS key that defines which principals can use the key and what cryptographic operations they may perform. Every customer-managed key must have exactly one key policy, and this policy is evaluated in combination with IAM policies to determine effective permissions.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/kms/latest/developerguide/key-policies.html

**Terrafrom Mappings:**

- `aws_kms_key_policy.key_id`

## Supported Methods

- `GET`: Get a KMS key policy by its Key ID
- ~~`LIST`~~
- `SEARCH`: Search KMS key policies by Key ID

## Possible Links

### [`kms-key`](/sources/aws/Types/kms-key)

A KMS key policy is attached to exactly one KMS key; this link represents that one-to-one relationship. Following the link from a policy to its `kms-key` will show the cryptographic key whose usage and management are governed by the policy.
