---
title: GCP Compute Machine Image
sidebar_label: gcp-compute-machine-image
---

A Google Cloud Compute Engine **Machine Image** is a first-class resource that stores all the information required to recreate one or more identical virtual machine instances, including boot and data disks, instance metadata, machine type, service accounts, and network interface definitions. Machine images make it easy to version-control complete VM templates and roll them out across projects or organisations.  
Official documentation: https://cloud.google.com/compute/docs/machine-images

**Terrafrom Mappings:**

- `google_compute_machine_image.name`

## Supported Methods

- `GET`: Get GCP Compute Machine Image by "gcp-compute-machine-image-name"
- `LIST`: List all GCP Compute Machine Image items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

The machine image contains snapshots of every persistent disk that was attached to the source VM. Linking a machine image to its underlying disks allows Overmind to surface risks such as outdated disk encryption keys or insufficient replication settings.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

A machine image is normally created from, or used to instantiate, Compute Engine instances. Tracking this relationship lets you see which VMs were the origin of the image and which new VMs will inherit its configuration or vulnerabilities.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Network interface settings embedded in the machine image reference specific VPC networks. Connecting the image to those networks helps identify issues like deprecated network configurations that new VMs would inherit.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Each network interface in the machine image also specifies a subnetwork. Mapping this linkage highlights potential problems such as subnet IP exhaustion or mismatched IAM policies that could affect any instance launched from the image.
