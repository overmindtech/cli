---
title: GCP Cloud Resource Manager Project
sidebar_label: gcp-cloud-resource-manager-project
---

A **Google Cloud Platform (GCP) Project** is the fundamental organising entity managed by the Cloud Resource Manager service. Every GCP workload—whether it is a single virtual machine or a complex, multi-region Kubernetes deployment—must reside inside a Project. The Project acts as a logical container for:

- All GCP resources (compute, storage, networking, databases, etc.)
- Identity and Access Management (IAM) policies
- Billing configuration
- Quotas and limits
- Metadata such as labels and organisation/folder hierarchy

Because policies and billing are enforced at the Project level, understanding the state of a Project is critical when assessing deployment risk. For detailed information, refer to the official Google documentation: https://cloud.google.com/resource-manager/docs/creating-managing-projects

## Supported Methods

- `GET`: Get a gcp-cloud-resource-manager-project by its "name"
- ~~`LIST`~~
- ~~`SEARCH`~~
