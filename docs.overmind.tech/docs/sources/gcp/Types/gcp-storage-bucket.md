---
title: GCP Storage Bucket
sidebar_label: gcp-storage-bucket
---

A GCP Storage Bucket is a logical container in Google Cloud Storage that holds your objects (blobs). Buckets provide globally-unique namespaces, configurable lifecycle policies, access controls, versioning, and encryption options, allowing organisations to store and serve unstructured data such as backups, media files, or static web assets. See the official documentation for full details: https://cloud.google.com/storage/docs/key-terms#buckets

**Terrafrom Mappings:**

- `google_storage_bucket.name`

## Supported Methods

- `GET`: Get a gcp-storage-bucket by its "name"
- `LIST`: List all gcp-storage-bucket
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Instances and other compute resources that run inside a VPC network often read from or write to a Storage Bucket. Additionally, when Private Google Access or VPC Service Controls are enabled, the bucket’s accessibility is governed by the associated compute network, creating a security dependency between the two resources.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

Audit logs for a Storage Bucket can be routed into a Cloud Logging bucket, and Logging buckets can export their contents to a Storage Bucket. Either configuration establishes a link whereby changes to the Storage Bucket may affect log retention and compliance.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Storage Bucket can be configured to use Customer-Managed Encryption Keys (CMEK). When this option is enabled, the bucket references a Cloud KMS CryptoKey for data-at-rest encryption, making the bucket’s availability and security reliant on the referenced key’s state and permissions.
