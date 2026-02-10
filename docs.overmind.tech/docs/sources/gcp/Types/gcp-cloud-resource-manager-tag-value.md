---
title: GCP Cloud Resource Manager Tag Value
sidebar_label: gcp-cloud-resource-manager-tag-value
---

A Tag Value is the value component of Google Cloud’s hierarchical tagging system, which allows you to attach fine-grained, policy-aware metadata to resources. Each Tag Value sits under a Tag Key and, together, the pair forms a tag that can be propagated across projects and folders within an organisation. Tags enable centralised governance, cost allocation, and conditional access control through IAM and Org Policy. For full details, see the official Google Cloud documentation: https://cloud.google.com/resource-manager/docs/tags/tags-creating-and-managing#tag-values

**Terrafrom Mappings:**

- `google_tags_tag_value.name`

## Supported Methods

- `GET`: Get a gcp-cloud-resource-manager-tag-value by its "name"
- ~~`LIST`~~
- `SEARCH`: Search for TagValues by TagKey.
