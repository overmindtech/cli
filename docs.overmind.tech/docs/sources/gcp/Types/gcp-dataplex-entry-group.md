---
title: GCP Dataplex Entry Group
sidebar_label: gcp-dataplex-entry-group
---

A Dataplex Entry Group is a logical container in Google Cloud that lives in the Data Catalog service and is used by Dataplex to organise metadata about datasets, tables and other data assets. By grouping related Data Catalog entries together, Entry Groups enable consistent discovery, governance and lineage tracking across lakes, zones and projects. Each Entry Group is created in a specific project and location and can be referenced by Dataplex jobs, policies and fine-grained IAM settings.  
For full details see Google’s REST reference: https://cloud.google.com/data-catalog/docs/reference/rest/v1/projects.locations.entryGroups

**Terrafrom Mappings:**

- `google_dataplex_entry_group.id`

## Supported Methods

- `GET`: Get a gcp-dataplex-entry-group by its "locations|entryGroups"
- ~~`LIST`~~
- `SEARCH`: Search for Dataplex entry groups in a location. Use the format "location" or "projects/[project_id]/locations/[location]/entryGroups/[entry_group_id]" which is supported for terraform mappings.
