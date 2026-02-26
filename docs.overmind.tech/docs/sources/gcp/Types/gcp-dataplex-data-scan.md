---
title: GCP Dataplex Data Scan
sidebar_label: gcp-dataplex-data-scan
---

A GCP Dataplex Data Scan is a first-class Dataplex resource that encapsulates the configuration and schedule for profiling data or validating data-quality rules against a registered asset such as a BigQuery table or files held in Cloud Storage. Each scan lives in a specific Google Cloud location and records its execution history, metrics and detected issues, allowing teams to understand data health before downstream workloads rely on it.  
For full details see the official REST reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.dataScans

**Terrafrom Mappings:**

  * `google_dataplex_datascan.id`

## Supported Methods

* `GET`: Get a gcp-dataplex-data-scan by its "locations|dataScans"
* ~~`LIST`~~
* `SEARCH`: Search for Dataplex data scans in a location. Use the location name e.g., 'us-central1' or the format "projects/[project_id]/locations/[location]/dataScans/[data_scan_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

A Dataplex Data Scan may target a BigQuery table as its data source; linking the scan to the table lets Overmind trace quality findings back to the exact table that will be affected by the deployment.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

When the data asset under review is a set of files stored in Cloud Storage, Dataplex references the underlying bucket. Linking the scan to the bucket reveals how changes to bucket configuration or contents could influence upcoming scan results.