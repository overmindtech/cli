---
title: GCP Compute Instance Group
sidebar_label: gcp-compute-instance-group
---

A Google Cloud Compute Instance Group is a logical collection of Virtual Machine (VM) instances running on Google Compute Engine that are treated as a single entity for deployment, scaling and load-balancing purposes. Instance groups can be managed (all VMs created from a common template and automatically kept in the desired size/state) or unmanaged (a user-assembled set of individual VMs). They are commonly used behind load balancers to provide highly available, horizontally scalable services.  
For full details see the official Google Cloud documentation: https://cloud.google.com/compute/docs/instance-groups

**Terrafrom Mappings:**

- `google_compute_instance_group.name`

## Supported Methods

- `GET`: Get GCP Compute Instance Group by "gcp-compute-instance-group-name"
- `LIST`: List all GCP Compute Instance Group items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every VM in an Instance Group must be attached to a VPC network. Overmind therefore links a Compute Instance Group to the Compute Network that provides its underlying connectivity, enabling you to trace how network-level policies or mis-configurations might affect the availability of the workload hosted by the group.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within a given VPC network, all VMs in the Instance Group reside in a specific subnetwork. Overmind links the Instance Group to that Subnetwork so you can understand IP address allocation, regional placement and any subnet-specific firewall rules that could impact the instances.
