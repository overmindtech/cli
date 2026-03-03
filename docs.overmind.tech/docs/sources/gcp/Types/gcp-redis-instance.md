---
title: GCP Redis Instance
sidebar_label: gcp-redis-instance
---

A GCP Redis Instance is a fully managed, in-memory data store provided by Cloud Memorystore for Redis. It offers a drop-in, highly available Redis service that handles provisioning, patching, scaling, monitoring and automatic fail-over, allowing you to use Redis as a cache or primary database without managing the underlying infrastructure yourself. See the official documentation for details: https://cloud.google.com/memorystore/docs/redis

**Terrafrom Mappings:**

- `google_redis_instance.id`

## Supported Methods

- `GET`: Get a gcp-redis-instance by its "locations|instances"
- ~~`LIST`~~
- `SEARCH`: Search Redis instances in a location. Use the format "location" or "projects/[project_id]/locations/[location]/instances/[instance_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled for the Redis instance, the data at rest is encrypted with a Cloud KMS Crypto Key. The Redis instance therefore depends on — and is cryptographically linked to — the specified `gcp-cloud-kms-crypto-key`.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A Redis instance is deployed inside a specific VPC network and is reachable only via an internal IP address in that network. Consequently, each instance is associated with a `gcp-compute-network`, which determines its connectivity and firewall boundaries.

### [`gcp-compute-ssl-certificate`](/sources/gcp/Types/gcp-compute-ssl-certificate)

When TLS is enabled for a Redis instance, it can reference a Compute Engine SSL certificate resource to present during encrypted client connections. The `gcp-compute-ssl-certificate` therefore represents the server certificate used to secure traffic to the Redis instance.
