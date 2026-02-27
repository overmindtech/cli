---
title: GCP Compute Machine Image
sidebar_label: gcp-compute-machine-image
---

A Google Cloud Compute Machine Image is a first-class resource that captures the full state of a virtual machine at a point in time, including all attached disks, metadata, instance properties, service-accounts, and network configuration. It can be used to recreate identical VMs quickly or share a golden template across projects and organisations. See the official documentation for full details: https://cloud.google.com/compute/docs/machine-images

**Terrafrom Mappings:**

* `google_compute_machine_image.name`

## Supported Methods

* `GET`: Get GCP Compute Machine Image by "gcp-compute-machine-image-name"
* `LIST`: List all GCP Compute Machine Image items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

A machine image may be protected with customer-managed encryption keys (CMEK); when this option is used it references the specific Cloud KMS Crypto Key Version that encrypts the image data.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

The boot disk and any additional data disks attached to the source instance are incorporated into the machine image. When a new instance is created from the machine image, new persistent disks are instantiated from these definitions.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

Within a machine image the boot disk is ultimately based on a Compute Image. Thus the machine image indirectly depends on, and records, the image that was used to build the source VM.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

A machine image is created from a source Compute Instance and can in turn be used to launch new instances that replicate the captured configuration.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Network interface settings, including the VPC network IDs, are stored in the machine image so that any VM instantiated from it can attach to the same or equivalent networks.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

Internally, Google Cloud may use snapshots of the instance’s disks when building the machine image. Conversely, users can export disks from a machine image as individual snapshots.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

The machine image stores the exact subnetwork configuration of each NIC, allowing recreated VMs to provision themselves in the same subnetworks.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Service accounts attached to the source instance are recorded in the machine image; any VM launched from the image inherits those service account bindings unless overridden.
