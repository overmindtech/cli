---
title: GCP Redis Instance
sidebar_label: gcp-redis-instance
---

Cloud Memorystore for Redis provides a fully managed, in-memory, open-source Redis service on Google Cloud. It is commonly used for low-latency caching, session management, real-time analytics and message brokering. When you create an instance Google handles provisioning, patching, monitoring, fail-over and, if requested, TLS encryption and customer-managed encryption keys (CMEK).  
More information can be found in the official documentation: https://cloud.google.com/memorystore/docs/redis

**Terrafrom Mappings:**

- `google_redis_instance.id`

## Supported Methods

- `GET`: Get a gcp-redis-instance by its "locations|instances"
- ~~`LIST`~~
- `SEARCH`: Search Redis instances in a location. Use the format "location" or "projects/[project_id]/locations/[location]/instances/[instance_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If CMEK is enabled, the Redis instance is encrypted at rest using a Cloud KMS CryptoKey. Overmind links the instance to the crypto key so you can trace data-at-rest encryption dependencies and evaluate key rotation or IAM policies.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Each Redis instance is created inside a specific VPC network and subnet. Linking to the compute network allows you to understand network reachability, firewall rules and peering arrangements that could affect the instance.

### [`gcp-compute-ssl-certificate`](/sources/gcp/Types/gcp-compute-ssl-certificate)

When TLS is enabled, Redis serves Google-managed certificates under the hood. Overmind associates these certificates (represented as Compute SSL Certificate resources) so that certificate expiry and chain of trust can be audited alongside the Redis service.
