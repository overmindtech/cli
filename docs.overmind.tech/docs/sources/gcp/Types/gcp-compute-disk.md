---
title: GCP Compute Disk
sidebar_label: gcp-compute-disk
---

A GCP Compute Disk is a durable, high-performance block-storage volume that can be attached to one or more Compute Engine virtual machine instances. Persistent disks can act as boot devices or as additional data volumes, are automatically replicated within a zone or region, and can be backed up through snapshots or turned into custom images for rapid redeployment.  
For full details see the official Google Cloud documentation: https://cloud.google.com/compute/docs/disks

**Terrafrom Mappings:**

- `google_compute_disk.name`

## Supported Methods

- `GET`: Get GCP Compute Disk by "gcp-compute-disk-name"
- `LIST`: List all GCP Compute Disk items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

This link appears when one persistent disk has been cloned or recreated from another (for example, using the `--source-disk` flag), allowing Overmind to follow ancestry or duplication chains between disks.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

A custom image may have been created from the current disk, or conversely the disk may have been created from an image. Overmind records this link so you can see which images depend on, or are the origin of, a particular disk.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Virtual machine instances to which the disk is attached (either as a boot disk or as an additional mounted volume) are linked here. This allows you to view the blast-radius of any change to the disk in terms of running workloads.

### [`gcp-compute-instant-snapshot`](/sources/gcp/Types/gcp-compute-instant-snapshot)

If an instant snapshot has been taken from the disk, or if the disk has been created from an instant snapshot, Overmind records the relationship via this link.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

Standard persistent disk snapshots derived from the disk, or snapshots that were used to create the disk, are linked here, enabling traceability between long-term backups and the live volume.
