---
title: GCP Compute Project
sidebar_label: gcp-compute-project
---

A Google Cloud Project is the fundamental organisational unit in Google Cloud Platform. It acts as a logical container for all your Google Cloud resources, identity and access management (IAM) policies, APIs, quotas and billing information. Every resource – from virtual machines to service accounts – is created in exactly one project, and project-level settings (such as audit logging, labels and network host project status) govern how those resources operate. See the official documentation for full details: https://cloud.google.com/resource-manager/docs/creating-managing-projects

**Terrafrom Mappings:**

- `google_project.project_id`
- `google_compute_shared_vpc_host_project.project`
- `google_compute_shared_vpc_service_project.service_project`
- `google_compute_shared_vpc_service_project.host_project`
- `google_project_iam_binding.project`
- `google_project_iam_member.project`
- `google_project_iam_policy.project`
- `google_project_iam_audit_config.project`

## Supported Methods

- `GET`: Get a gcp-compute-project by its "name"
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Service accounts are identities that live inside a project. Overmind links a gcp-iam-service-account to its parent gcp-compute-project to show which project owns and governs the credentials and IAM permissions of that service account.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Every Cloud Storage bucket is created within a specific project. Overmind establishes a link from a gcp-storage-bucket back to its gcp-compute-project so you can trace ownership, billing and IAM inheritance for the bucket.
