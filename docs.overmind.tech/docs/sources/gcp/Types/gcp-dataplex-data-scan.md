---
title: GCP Dataplex Data Scan
sidebar_label: gcp-dataplex-data-scan
---

A Dataplex Data Scan is a managed resource that schedules and executes automated profiling or data-quality checks over data held in Google Cloud Platform (GCP) storage systems such as Cloud Storage and BigQuery. The scan stores its configuration, execution history and results, allowing teams to understand the structure, completeness and validity of their datasets before those datasets are used downstream. Full details can be found in the official Google Cloud documentation: https://docs.cloud.google.com/dataplex/docs/use-data-profiling

**Terrafrom Mappings:**

- `google_dataplex_datascan.id`

## Supported Methods

- `GET`: Get a gcp-dataplex-data-scan by its "locations|dataScans"
- ~~`LIST`~~
- `SEARCH`: Search for Dataplex data scans in a location. Use the location name e.g., 'us-central1' or the format "projects/[project_id]/locations/[location]/dataScans/[data_scan_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

A Dataplex Data Scan can target objects stored in a Cloud Storage bucket for profiling or quality validation. Therefore, Overmind links the scan resource to the bucket that contains the underlying data being analysed, enabling a complete view of the data-quality pipeline and its dependencies.
