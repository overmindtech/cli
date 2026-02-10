---
title: KMS Alias
sidebar_label: kms-alias
---

An AWS Key Management Service (KMS) alias is a human-readable pointer to a specific KMS key, allowing you to reference that key without exposing its full KeyID or ARN. Aliases make it simpler to rotate keys and update applications, because you can move the alias to a new key rather than changing code or configurations that use the key directly. They are unique within an account and region, and can reference either customer-managed or AWS-managed keys.  
For further details, see the official AWS documentation: https://docs.aws.amazon.com/kms/latest/developerguide/kms-alias.html

**Terrafrom Mappings:**

- `aws_kms_alias.arn`

## Supported Methods

- `GET`: Get an alias by keyID/aliasName
- `LIST`: List all aliases
- `SEARCH`: Search aliases by keyID

## Possible Links

### [`kms-key`](/sources/aws/Types/kms-key)

Each alias is a shorthand reference that maps to exactly one KMS key; the link shows which underlying `kms-key` the alias currently points to, enabling you to trace risk and usage back to the actual cryptographic key material.
