---
title: GCP Compute Firewall
sidebar_label: gcp-compute-firewall
---

A GCP Compute Firewall is a set of rules that control incoming and outgoing network traffic to Virtual Machine (VM) instances within a Google Cloud Virtual Private Cloud (VPC) network. Each rule defines whether specific connections (identified by protocol, port, source, destination and direction) are allowed or denied, thereby providing network-level security and segmentation for workloads running on Google Cloud.

**Terrafrom Mappings:**

- `google_compute_firewall.name`

## Supported Methods

- `GET`: Get a gcp-compute-firewall by its "name"
- `LIST`: List all gcp-compute-firewall
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A firewall rule is always created inside a single VPC network; that network determines the scope within which the rule is evaluated. Overmind therefore links a gcp-compute-firewall to the gcp-compute-network that owns it.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Firewall rules can specify target or source service accounts, allowing traffic to be filtered based on the workload identity running on a VM. Overmind links the firewall rule to any gcp-iam-service-account referenced in its `target_service_accounts` or `source_service_accounts` fields.
