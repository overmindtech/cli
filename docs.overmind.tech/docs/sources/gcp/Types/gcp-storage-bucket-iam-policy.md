---
title: GCP Storage Bucket Iam Policy
sidebar_label: gcp-storage-bucket-iam-policy
---

A **Storage Bucket IAM policy** defines who (principals) can perform which actions (roles/permissions) on a specific Cloud Storage bucket. It is the fine-grained access-control object that sits on top of a bucket and overrides or complements broader project-level IAM settings. For full details, see the Google Cloud documentation: https://cloud.google.com/storage/docs/access-control/iam

**Terrafrom Mappings:**

* `google_storage_bucket_iam_binding.bucket`
* `google_storage_bucket_iam_member.bucket`
* `google_storage_bucket_iam_policy.bucket`

## Supported Methods

* `GET`: Get GCP Storage Bucket Iam Policy by "gcp-storage-bucket-iam-policy-bucket"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Storage Bucket Iam Policy by "gcp-storage-bucket-iam-policy-bucket"

## Possible Links

### [`gcp-compute-project`](/sources/gcp/Types/gcp-compute-project)

The bucket IAM policy is scoped within a single GCP project; therefore every policy item is linked back to the project that owns the bucket.

### [`gcp-iam-role`](/sources/gcp/Types/gcp-iam-role)

Each binding inside the policy references one or more IAM roles that grant permissions; this link shows which predefined or custom roles are in use.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Service accounts are common principals in bucket policies. Linking reveals which service accounts have been granted access and with what privileges.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

The IAM policy is attached to and governs a specific Cloud Storage bucket; this link connects the policy object to the underlying bucket resource.
