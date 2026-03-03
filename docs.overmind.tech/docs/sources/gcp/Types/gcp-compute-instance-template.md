---
title: GCP Compute Instance Template
sidebar_label: gcp-compute-instance-template
---

A Google Cloud Compute Instance Template is a reusable description of the properties required to create a virtual machine (VM) instance. It encapsulates details such as machine type, boot image, disks, network interfaces, metadata, tags, and service-account settings. Once defined, the template can be used by users, managed instance groups, autoscalers, or other automation to create identically configured VMs at scale.  
Official documentation: https://cloud.google.com/compute/docs/instance-templates

**Terrafrom Mappings:**

- `google_compute_instance_template.name`

## Supported Methods

- `GET`: Get a gcp-compute-instance-template by its "name"
- `LIST`: List all gcp-compute-instance-template
- `SEARCH`: Search for instance templates by network tag. The query is a plain network tag name.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

An instance template can reference a customer-managed encryption key (CMEK) from Cloud KMS to encrypt the persistent disks defined in the template.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

Boot and additional persistent disks are specified inside the template. Any disk image or snapshot expanded into an actual persistent disk at instance-creation time will appear as a linked compute-disk resource.

### [`gcp-compute-firewall`](/sources/gcp/Types/gcp-compute-firewall)

The network tags set in the template are used by VMs launched from it. Firewall rules that target those tags therefore become effective for every instance derived from the template.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

The template’s boot disk references a specific compute image (public, custom, or shared). This image is the source from which the VM’s root filesystem is created.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

When a VM is launched using this template—either manually or by a managed instance group—the resulting resource is a compute-instance that maintains a provenance link back to the template.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Each network interface declared in the template must point to a VPC network, establishing the connectivity context for all future instances based on the template.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If node affinity is configured in the template, instances created from it will attempt to schedule onto the specified sole-tenant node group.

### [`gcp-compute-reservation`](/sources/gcp/Types/gcp-compute-reservation)

A template can include reservation affinity, causing newly created VMs to consume capacity from a specific Compute Engine reservation.

### [`gcp-compute-route`](/sources/gcp/Types/gcp-compute-route)

Although routes are defined at the network level, all VMs derived from the template inherit those routes through their attached network, so routing behaviour is indirectly influenced by the template.

### [`gcp-compute-security-policy`](/sources/gcp/Types/gcp-compute-security-policy)

If instances launched from the template are later attached to backend services that use Cloud Armor security policies, their traffic will be evaluated against those policies; tracing the link helps assess exposure.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

The template may specify a source snapshot instead of an image for one or more disks, resulting in disks that are restored from those snapshots at VM creation time.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

For each network interface, the template can identify a specific subnetwork, dictating the IP range from which the instance will draw its primary internal address.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

A service account can be attached in the template so that every VM started from it runs with the same IAM identity and associated OAuth scopes.
