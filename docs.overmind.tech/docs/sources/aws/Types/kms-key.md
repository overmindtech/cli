---
title: KMS Key
sidebar_label: kms-key
---

An AWS Key Management Service (KMS) Key is a logical representation of a cryptographic key used to encrypt and decrypt data across AWS services and your own applications. Each key is uniquely identifiable by its Key ID and Amazon Resource Name (ARN), can be either customer-managed or AWS-managed, and is stored within an AWS-managed hardware security module (HSM) cluster or, when using a custom key store, in an AWS CloudHSM cluster that you control. KMS Keys are central to implementing envelope encryption, controlling access to encrypted resources, and meeting compliance requirements related to data protection.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html

**Terrafrom Mappings:**

- `aws_kms_key.key_id`

## Supported Methods

- `GET`: Get a KMS Key by its ID
- `LIST`: List all KMS Keys
- `SEARCH`: Search for KMS Keys by ARN

## Possible Links

### [`kms-custom-key-store`](/sources/aws/Types/kms-custom-key-store)

A KMS Key may reside in a custom key store backed by your own AWS CloudHSM cluster. This link is produced when the key’s `KeyStoreId` attribute is set, allowing Overmind to trace the relationship between the key and the custom key store that physically holds its material.

### [`kms-key-policy`](/sources/aws/Types/kms-key-policy)

Every KMS Key has exactly one key policy that defines which principals are authorised to use or administer the key. Overmind links a key to its policy so that you can quickly inspect who can access the key and identify potential misconfigurations or excessive permissions.

### [`kms-grant`](/sources/aws/Types/kms-grant)

Grants provide time-bound or scoped permissions for principals to use a KMS Key without modifying its key policy. Overmind records links from a key to all active grants, enabling you to see what temporary or delegated access exists and to assess the risk of unintended key usage.
