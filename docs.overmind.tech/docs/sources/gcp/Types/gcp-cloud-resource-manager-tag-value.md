---
title: GCP Cloud Resource Manager Tag Value
sidebar_label: gcp-cloud-resource-manager-tag-value
---

A GCP Cloud Resource Manager **Tag Value** is the second layer in Google Cloud’s new tagging hierarchy, sitting beneath a Tag Key and above the individual resources to which it is applied. Together, Tag Keys and Tag Values allow administrators to attach fine-grained, organisation-wide metadata to projects, folders and individual cloud resources, enabling consistent policy enforcement, cost allocation, automation and reporting across an estate. Each Tag Value represents a specific, permitted value for a given Tag Key (e.g. Tag Key `environment` may have Tag Values `production`, `staging`, `test`).  
For a full description of Tag Values and how they fit into the tagging system, refer to Google’s documentation: https://cloud.google.com/resource-manager/reference/rest/v3/tagValues.

**Terrafrom Mappings:**

- `google_tags_tag_value.name`

## Supported Methods

- `GET`: Get a gcp-cloud-resource-manager-tag-value by its "name"
- ~~`LIST`~~
- `SEARCH`: Search for TagValues by TagKey.
