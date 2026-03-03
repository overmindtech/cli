---
title: GCP Compute Regional Instance Group Manager
sidebar_label: gcp-compute-regional-instance-group-manager
---

A Google Cloud Compute Regional Instance Group Manager (RIGM) is a control plane resource that creates, deletes, updates and monitors a homogeneous set of virtual machine (VM) instances that are distributed across two or more zones within the same region. By using a RIGM you gain automated rolling updates, proactive auto-healing and the ability to spread workload across zones for higher availability.  
Official documentation: https://cloud.google.com/compute/docs/instance-groups/creating-groups-of-managed-instances#regional

**Terrafrom Mappings:**

- `google_compute_region_instance_group_manager.name`

## Supported Methods

- `GET`: Get GCP Compute Regional Instance Group Manager by "gcp-compute-regional-instance-group-manager-name"
- `LIST`: List all GCP Compute Regional Instance Group Manager items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-autoscaler`](/sources/gcp/Types/gcp-compute-autoscaler)

A regional instance group manager can be linked to an Autoscaler resource that dynamically adjusts the number of VM instances in the managed group based on load, schedules or custom metrics.

### [`gcp-compute-health-check`](/sources/gcp/Types/gcp-compute-health-check)

Health checks are referenced by the RIGM to perform auto-healing; instances that fail the configured health check are recreated automatically.

### [`gcp-compute-instance-group`](/sources/gcp/Types/gcp-compute-instance-group)

The RIGM creates and controls a Regional Managed Instance Group. This underlying instance group is where the actual VM instances live and where traffic is balanced.

### [`gcp-compute-instance-template`](/sources/gcp/Types/gcp-compute-instance-template)

Every RIGM points to an Instance Template that defines the machine type, boot disk, metadata and other properties used when new VM instances are instantiated.

### [`gcp-compute-target-pool`](/sources/gcp/Types/gcp-compute-target-pool)

For legacy network load balancing, a RIGM can register its instances with a Target Pool so that traffic from a network load balancer is distributed across the managed instances.
