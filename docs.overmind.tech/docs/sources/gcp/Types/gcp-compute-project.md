---
title: GCP Compute Project
sidebar_label: gcp-compute-project
---

A Google Cloud project is the top-level, logical container for every resource you create in Google Cloud. It stores metadata such as billing configuration, IAM policy, APIs that are enabled, default network settings and quotas, and it provides an isolated namespace for resource names. In the context of Compute Engine, the project determines which VM instances, disks, firewalls and other compute resources can interact, and it is the unit against which most permissions and quotas are enforced.  
Official documentation: https://cloud.google.com/resource-manager/docs/creating-managing-projects

## Supported Methods

- `GET`: Get a gcp-compute-project by its "name"
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every service account is created inside a single project and inherits that project’s IAM policy unless overridden. Overmind links a `gcp-compute-project` to the `gcp-iam-service-account` resources it owns so that you can trace how credentials and permissions propagate within the project.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets live inside a project and consume that project’s quotas and billing account. Linking a `gcp-compute-project` to its `gcp-storage-bucket` resources lets you see which data stores are affected by changes to project-wide settings such as IAM roles or organisation policies.
