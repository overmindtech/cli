---
title: Custom Key Store
sidebar_label: kms-custom-key-store
---

A custom key store in AWS Key Management Service (KMS) enables you to back your KMS keys with your own AWS CloudHSM cluster rather than with the default, multi-tenant KMS hardware security modules. This gives you exclusive control over the cryptographic hardware that protects your key material while still allowing you to use KMS APIs and integrations. You can create, connect, disconnect, or delete a custom key store, and any KMS keys that reside in it remain under your sole tenancy. See the official AWS documentation for full details: https://docs.aws.amazon.com/kms/latest/developerguide/custom-key-store-overview.html

**Terrafrom Mappings:**

- `aws_kms_custom_key_store.id`

## Supported Methods

- `GET`: Get a custom key store by its ID
- `LIST`: List all custom key stores
- `SEARCH`: Search custom key store by ARN
