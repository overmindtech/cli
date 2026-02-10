---
title: GCP Compute Instance Template
sidebar_label: gcp-compute-instance-template
---

A Compute Engine instance template is a reusable blueprint that captures almost all of the configuration needed to launch a Virtual Machine (VM) instance in Google Cloud: machine type, boot image, attached disks, network interfaces, metadata, service accounts, shielded-VM options and more. Templates allow you to create individual VM instances consistently or serve as the basis for managed instance groups that can scale automatically.  
Official documentation: https://cloud.google.com/compute/docs/instance-templates

**Terrafrom Mappings:**

- `google_compute_instance_template.name`

## Supported Methods

- `GET`: Get a gcp-compute-instance-template by its "name"
- `LIST`: List all gcp-compute-instance-template
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption keys (CMEK) are specified in the template, they reference a Cloud KMS crypto-key that will be used to encrypt the boot or data disks of any VM created from the template.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

The template can define additional persistent disks to be auto-created and attached, or it can attach existing disks in read-only or read-write mode.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

The boot disk section of the template points to a Compute Engine image that is cloned each time a new VM is launched.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

When a user or an autoscaler instantiates the template, it materialises as one or more Compute Engine instances that inherit every property defined in the template.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every network interface defined in the template must belong to a VPC network, so the template contains links to the relevant network resources.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If the template targets sole-tenant nodes, it can specify a node group affinity so that all created VMs land on a particular node group.

### [`gcp-compute-reservation`](/sources/gcp/Types/gcp-compute-reservation)

Templates may be configured to consume capacity from an existing reservation, ensuring launched VMs fit within reserved resources.

### [`gcp-compute-security-policy`](/sources/gcp/Types/gcp-compute-security-policy)

Tags or service-account settings in the template can cause the resulting instances to match Cloud Armor security policies applied at the project or network level.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

Instead of an image, the template can build new disks from a snapshot, linking the template to that snapshot resource.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

For networks that are in auto or custom subnet mode, the template points to the exact subnetwork each NIC should join.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The template includes a service account and its OAuth scopes; the created VMs will assume that service account’s identity and permissions.
