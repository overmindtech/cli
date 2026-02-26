---
title: GCP Compute Instant Snapshot
sidebar_label: gcp-compute-instant-snapshot
---

A GCP Compute Instant Snapshot is a point-in-time, crash-consistent copy of a Compute Engine persistent disk that is created almost instantaneously, permitting rapid backup, cloning, and disaster-recovery workflows. Instant snapshots can be used to restore a disk to the exact state it was in when the snapshot was taken or to create new disks that replicate that state. They differ from traditional snapshots primarily in the speed at which they are taken and restored.  
Official documentation: https://cloud.google.com/compute/docs/disks/instant-snapshots

**Terrafrom Mappings:**

  * `google_compute_instant_snapshot.name`

## Supported Methods

* `GET`: Get GCP Compute Instant Snapshot by "gcp-compute-instant-snapshot-name"
* `LIST`: List all GCP Compute Instant Snapshot items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

An instant snapshot is always sourced from an existing Compute Engine persistent disk. Therefore, each `gcp-compute-instant-snapshot` has a direct parent–child relationship with the `gcp-compute-disk` it captures, and Overmind links the snapshot back to the originating disk to surface dependency and recovery paths.