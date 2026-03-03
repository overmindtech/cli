---
title: GCP Storage Bucket
sidebar_label: gcp-storage-bucket
---

A Google Cloud Storage Bucket is a globally-unique container used to store, organise and serve objects (files) in Google Cloud Storage. Buckets provide configuration points for data location, access control, lifecycle management, encryption and logging. They are the fundamental resource for object storage workloads such as static website hosting, backup, or data lakes.  
For full details see the official documentation: https://cloud.google.com/storage/docs/buckets

**Terrafrom Mappings:**

- `google_storage_bucket.name`
- `google_storage_bucket_iam_binding.bucket`
- `google_storage_bucket_iam_member.bucket`
- `google_storage_bucket_iam_policy.bucket`

## Supported Methods

- `GET`: Get a gcp-storage-bucket by its "name"
- `LIST`: List all gcp-storage-bucket
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A bucket may be encrypted with a customer-managed encryption key (CMEK) that resides in Cloud KMS. The bucket’s encryption configuration therefore references the corresponding `gcp-cloud-kms-crypto-key`.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When VPC Service Controls or Private Google Access are used, access between a Compute Network and a Storage Bucket is constrained or allowed based on network settings. Log sinks from VPC flow logs can also target a Storage Bucket, creating a relationship between the bucket and the originating `gcp-compute-network`.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

Cloud Logging can route logs from a Logging Bucket to Cloud Storage for long-term retention or auditing. If such a sink targets this Storage Bucket, the bucket becomes linked to the source `gcp-logging-bucket`.

### [`gcp-storage-bucket-iam-policy`](/sources/gcp/Types/gcp-storage-bucket-iam-policy)

Every Storage Bucket has an IAM policy that defines who can read, write or administer it. That policy is exposed as a separate `gcp-storage-bucket-iam-policy` object, which is directly attached to this bucket.
