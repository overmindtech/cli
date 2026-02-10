---
title: GCP Compute Instant Snapshot
sidebar_label: gcp-compute-instant-snapshot
---

A GCP Compute Instant Snapshot is a point-in-time, crash-consistent copy of a persistent disk that is captured almost immediately, irrespective of the size of the disk. It is stored in the same region as the source disk and is intended for rapid backup, testing, or disaster-recovery scenarios where minimal creation time is essential. Instant snapshots are ephemeral by design (they are automatically deleted after seven days unless converted to a regular snapshot) and incur lower network egress because the data never leaves the region.  
For full details, refer to the official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/instantSnapshots

**Terrafrom Mappings:**

- `google_compute_instant_snapshot.name`

## Supported Methods

- `GET`: Get GCP Compute Instant Snapshot by "gcp-compute-instant-snapshot-name"
- `LIST`: List all GCP Compute Instant Snapshot items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

An Instant Snapshot is created from a persistent disk. The snapshot’s `source_disk` field references the original `gcp-compute-disk`, and any restore or promotion operation will require access to that underlying disk or its region.
