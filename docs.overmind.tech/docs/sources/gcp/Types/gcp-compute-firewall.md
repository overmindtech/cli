---
title: GCP Compute Firewall
sidebar_label: gcp-compute-firewall
---

A Google Cloud VPC firewall rule controls inbound and outbound traffic to and from the virtual machine (VM) instances that are attached to a particular VPC network. Each rule specifies a direction, priority, action (allow or deny), protocol and port list, and a target (network tags or service accounts). Rules are stateful and are evaluated before traffic reaches any instance, allowing you to centrally enforce network security policy across your workloads.  
Official documentation: https://cloud.google.com/vpc/docs/firewalls

**Terrafrom Mappings:**

  * `google_compute_firewall.name`

## Supported Methods

* `GET`: Get a gcp-compute-firewall by its "name"
* `LIST`: List all gcp-compute-firewall
* `SEARCH`: Search for firewalls by network tag. The query is a plain network tag name.

## Possible Links

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Firewall rules apply to VM instances that match their target criteria (network tags or service accounts). Therefore, an instance is linked to the firewall rules that currently govern the traffic it may send or receive.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every firewall rule is created within a specific VPC network. The rule only affects resources that are attached to that network, so it is linked to its parent network resource.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Firewall rules can target VM instances by the service account they are running as. When a rule uses the `target_service_accounts` field, it is related to those IAM service accounts.