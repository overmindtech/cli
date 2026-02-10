---
title: GCP Compute Instance
sidebar_label: gcp-compute-instance
---

A GCP Compute Instance is a virtual machine (VM) hosted on Google Cloud’s Compute Engine service. It provides configurable CPU, memory, storage and operating-system options, enabling you to run anything from small test services to large-scale production workloads. Instances can be created from public images or custom images, can have one or more network interfaces, and can attach multiple persistent or ephemeral disks. For full details see the official documentation: https://cloud.google.com/compute/docs/instances

**Terrafrom Mappings:**

- `google_compute_instance.name`

## Supported Methods

- `GET`: Get GCP Compute Instance by "gcp-compute-instance-name"
- `LIST`: List all GCP Compute Instance items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

A Compute Instance normally boots from and/or mounts one or more persistent disks. Overmind links an instance to every `gcp-compute-disk` that is attached to it so you can assess the impact of changes to those disks on the VM.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every network interface on a Compute Instance is connected to a VPC network. Overmind records this relationship to show how altering a `gcp-compute-network` (for example, changing routing or firewall rules) could affect the instance’s connectivity.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within a VPC network, an interface resides in a specific subnetwork. Overmind links the instance to its `gcp-compute-subnetwork` so you can evaluate risks related to IP ranges, regional availability or subnet-level security policies that might influence the VM.
