---
title: GCP Compute Instance
sidebar_label: gcp-compute-instance
---

A Google Cloud Compute Engine instance is a virtual machine (VM) that runs on Google’s infrastructure. It provides configurable CPU, memory, disk and network resources so you can run workloads in a scalable, on-demand manner. For full details see the official documentation: https://cloud.google.com/compute/docs/instances.

**Terrafrom Mappings:**

* `google_compute_instance.name`

## Supported Methods

* `GET`: Get GCP Compute Instance by "gcp-compute-instance-name"
* `LIST`: List all GCP Compute Instance items
* `SEARCH`: Search for GCP Compute Instance by "gcp-compute-instance-networkTag"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the instance’s boot or data disks are encrypted with customer-managed encryption keys (CMEK), it references a Cloud KMS crypto key.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

A specific version of the KMS key may be recorded when CMEK encryption is enabled on the instance’s disks.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

Boot and additional persistent disks are attached to the instance; these disks back the VM’s storage.

### [`gcp-compute-firewall`](/sources/gcp/Types/gcp-compute-firewall)

Firewall rules that target the instance’s network tags or service account control inbound and outbound traffic for the VM.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

The instance’s boot disk is created from a Compute Engine image, capturing the operating system and initial state.

### [`gcp-compute-instance-group-manager`](/sources/gcp/Types/gcp-compute-instance-group-manager)

When the VM is part of a managed instance group (MIG), the group manager is responsible for creating, deleting and updating the instance.

### [`gcp-compute-instance-template`](/sources/gcp/Types/gcp-compute-instance-template)

Instances launched via a template inherit machine type, disks, metadata and network settings defined in that template.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every network interface on the instance is connected to a VPC network, determining the VM’s reachable address space.

### [`gcp-compute-route`](/sources/gcp/Types/gcp-compute-route)

Routes in the attached VPC network dictate how the instance’s traffic is forwarded; some routes may apply only to instances with specific tags.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

Snapshots can be taken from the instance’s persistent disks for backup or cloning purposes, creating a link between the VM and its snapshots.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Each network interface is placed within a subnetwork, assigning the instance its internal IP range and regional scope.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

An optional service account is attached to the instance, granting it IAM-scoped credentials to access Google APIs.
