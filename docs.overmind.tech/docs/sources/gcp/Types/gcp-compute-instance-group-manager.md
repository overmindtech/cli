---
title: GCP Compute Instance Group Manager
sidebar_label: gcp-compute-instance-group-manager
---

A Compute Instance Group Manager (IGM) is the control plane object for a Managed Instance Group in Google Cloud Platform. It is responsible for creating, deleting, and maintaining a homogeneous fleet of Compute Engine virtual machines according to a declarative configuration such as target size, instance template and update policy. Because the manager continually reconciles the group’s actual state with the desired state, it underpins features like rolling updates, auto-healing and autoscaling.  
Official documentation: https://cloud.google.com/compute/docs/instance-groups/creating-groups-of-managed-instances

**Terrafrom Mappings:**

* `google_compute_instance_group_manager.name`

## Supported Methods

* `GET`: Get GCP Compute Instance Group Manager by "gcp-compute-instance-group-manager-name"
* `LIST`: List all GCP Compute Instance Group Manager items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-autoscaler`](/sources/gcp/Types/gcp-compute-autoscaler)

An Autoscaler resource can target a Managed Instance Group via its Instance Group Manager, dynamically increasing or decreasing the group’s size based on utilisation metrics or schedules.

### [`gcp-compute-health-check`](/sources/gcp/Types/gcp-compute-health-check)

Within an auto-healing policy the Instance Group Manager references one or more Health Check resources to decide when individual instances should be recreated.

### [`gcp-compute-instance-group`](/sources/gcp/Types/gcp-compute-instance-group)

The Instance Group Manager encapsulates and manages an underlying (managed) Instance Group resource that represents the actual collection of VM instances.

### [`gcp-compute-instance-template`](/sources/gcp/Types/gcp-compute-instance-template)

The manager uses an Instance Template to define the configuration (machine type, disks, metadata, etc.) of every VM it creates in the group.

### [`gcp-compute-target-pool`](/sources/gcp/Types/gcp-compute-target-pool)

For legacy network load balancing, an Instance Group Manager can be configured to automatically add or remove its instances from a Target Pool, enabling them to receive traffic from a forwarding rule.
