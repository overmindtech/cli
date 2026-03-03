---
title: GCP Dataplex Aspect Type
sidebar_label: gcp-dataplex-aspect-type
---

A Dataplex Aspect Type is a top-level resource within Google Cloud Dataplex’s metadata service that defines the structure of a metadata “aspect” – a reusable schema describing a set of attributes you want to attach to data assets (for example, data quality scores or business classifications). Once an aspect type is created, individual assets such as tables, files or columns can be annotated with concrete “aspects” that conform to that schema, ensuring consistent, centrally-governed metadata across your lake.  
For further details see the official API reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.aspectTypes

**Terrafrom Mappings:**

- `google_dataplex_aspect_type.id`

## Supported Methods

- `GET`: Get a gcp-dataplex-aspect-type by its "locations|aspectTypes"
- ~~`LIST`~~
- `SEARCH`: Search for Dataplex aspect types in a location. Use the format "location" or "projects/[project_id]/locations/[location]/aspectTypes/[aspect_type_id]" which is supported for terraform mappings.
