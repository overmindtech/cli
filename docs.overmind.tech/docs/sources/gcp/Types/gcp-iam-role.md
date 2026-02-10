---
title: GCP Iam Role
sidebar_label: gcp-iam-role
---

Google Cloud Identity and Access Management (IAM) roles are collections of granular permissions that you grant to principals—such as users, groups or service accounts—so they can interact with Google Cloud resources. Roles come in three varieties (basic, predefined and custom) and are the chief mechanism for enforcing the principle of least privilege across your estate. Overmind represents each IAM role as an individual resource, enabling you to surface the blast-radius of creating, modifying or deleting a role before you commit the change.  
For further details, refer to the official Google Cloud documentation: https://cloud.google.com/iam/docs/understanding-roles

## Supported Methods

- `GET`: Get a gcp-iam-role by its "name"
- `LIST`: List all gcp-iam-role
- ~~`SEARCH`~~
