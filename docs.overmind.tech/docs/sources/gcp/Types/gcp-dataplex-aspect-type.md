---
title: GCP Dataplex Aspect Type
sidebar_label: gcp-dataplex-aspect-type
---

A Google Cloud Dataplex Aspect Type is a reusable template that describes the structure and semantics of a particular piece of metadata—an _aspect_—that can later be attached to Dataplex assets, entries, or partitions. By defining aspect types centrally, an organisation can guarantee that the same metadata schema (for example, “Personally Identifiable Information classification” or “Data-quality score”) is applied consistently across lakes, zones, and assets, thereby strengthening governance, lineage, and discovery capabilities.  
For further details, see the official Dataplex REST reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.aspectTypes

**Terrafrom Mappings:**

- `google_dataplex_aspect_type.id`

## Supported Methods

- `GET`: Get a gcp-dataplex-aspect-type by its "locations|aspectTypes"
- ~~`LIST`~~
- `SEARCH`: Search for Dataplex aspect types in a location. Use the format "location" or "projects/[project_id]/locations/[location]/aspectTypes/[aspect_type_id]" which is supported for terraform mappings.
