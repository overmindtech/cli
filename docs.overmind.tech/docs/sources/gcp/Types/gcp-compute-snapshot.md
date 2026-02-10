---
title: GCP Compute Snapshot
sidebar_label: gcp-compute-snapshot
---

A GCP Compute Snapshot is a point-in-time, incremental backup of a Compute Engine persistent disk. Snapshots allow you to restore data following accidental deletion, corruption, or regional outage, and can also be used to create new disks in the same or a different project/region. Because snapshots are incremental, only the blocks that have changed since the last snapshot are stored, reducing cost and network egress. Snapshots can be scheduled, encrypted with customer-managed keys, and shared across projects through Cloud Storage-backed snapshot storage.  
Official documentation: https://cloud.google.com/compute/docs/disks/snapshots

**Terrafrom Mappings:**

- `google_compute_snapshot.name`

## Supported Methods

- `GET`: Get GCP Compute Snapshot by "gcp-compute-snapshot-name"
- `LIST`: List all GCP Compute Snapshot items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

A snapshot is created from a specific persistent disk; the link lets you trace a snapshot back to the disk it protects, or discover all snapshots derived from that disk.

### [`gcp-compute-instant-snapshot`](/sources/gcp/Types/gcp-compute-instant-snapshot)

An instant snapshot can later be converted into a standard snapshot, or serve as an intermediary during a snapshot operation. This link shows lineage between an instant snapshot and the resulting persistent snapshot resource.
