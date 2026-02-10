---
title: GCP Compute Instance Group
sidebar_label: gcp-compute-instance-group
---

A Google Cloud Compute Instance Group is a logical collection of virtual machine (VM) instances that you manage as a single entity. Instance groups can be either managed (where the group is tied to an instance template and can perform auto-healing, autoscaling and rolling updates) or unmanaged (a simple grouping of individually created VMs). They are commonly used to distribute traffic across identical instances and to simplify operational tasks such as scaling and updates.  
For an in-depth explanation, refer to the official documentation: https://cloud.google.com/compute/docs/instance-groups

**Terrafrom Mappings:**

- `google_compute_instance_group.name`

## Supported Methods

- `GET`: Get GCP Compute Instance Group by "gcp-compute-instance-group-name"
- `LIST`: List all GCP Compute Instance Group items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Each VM contained in the instance group is attached to a specific VPC network. Consequently, the instance group inherits a dependency on that GCP Compute Network; changes to the network (e.g., firewall rules, routing) can directly impact the availability or behaviour of all instances in the group.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within its parent VPC network, every instance is placed in a particular subnetwork. Therefore, the instance group is transitively linked to the associated GCP Compute Subnetwork. Subnetwork configuration—such as IP ranges or regional placement—affects how the grouped instances communicate internally and with external resources.
