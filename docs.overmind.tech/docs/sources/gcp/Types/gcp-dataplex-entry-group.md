---
title: GCP Dataplex Entry Group
sidebar_label: gcp-dataplex-entry-group
---

A Dataplex Entry Group is a logical container that holds one or more metadata entries within Google Cloud’s unified Data Catalog. By grouping related entries together, it helps data stewards organise, secure and search metadata that describe the underlying data assets managed by Dataplex (such as tables, files or streams). Each Entry Group lives in a specific project and location and can be granted IAM permissions independently, allowing fine-grained access control over the metadata it contains.  
Official documentation: https://cloud.google.com/data-catalog/docs/reference/rest/v1/projects.locations.entryGroups

**Terrafrom Mappings:**

  * `google_dataplex_entry_group.id`

## Supported Methods

* `GET`: Get a gcp-dataplex-entry-group by its "locations|entryGroups"
* ~~`LIST`~~
* `SEARCH`: Search for Dataplex entry groups in a location. Use the format "location" or "projects/[project_id]/locations/[location]/entryGroups/[entry_group_id]" which is supported for terraform mappings.