---
title: GCP Compute Forwarding Rule
sidebar_label: gcp-compute-forwarding-rule
---

A GCP Compute Forwarding Rule defines how incoming packets are directed within Google Cloud. It associates an IP address, protocol and port range with a specific target—such as a load-balancer target proxy, VPN gateway, or, for certain internal load-balancer variants, a backend service—so that traffic is forwarded correctly. Forwarding rules can be global or regional and, when internal, are bound to a particular VPC network (and optionally a subnetwork) to control the scope of traffic distribution.
For full details see the official documentation: https://docs.cloud.google.com/load-balancing/docs/forwarding-rule-concepts

**Terrafrom Mappings:**

- `google_compute_forwarding_rule.name`

## Supported Methods

- `GET`: Get GCP Compute Forwarding Rule by "gcp-compute-forwarding-rule-name"
- `LIST`: List all GCP Compute Forwarding Rule items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

For certain internal load balancers (e.g. Internal TCP/UDP Load Balancer), the forwarding rule points directly to a backend service. Overmind records this as a link so that any risk identified on the backend service can be surfaced when assessing the forwarding rule.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

An internal forwarding rule is created inside a specific VPC network; the rule determines how traffic is routed within that network. Linking the forwarding rule to its VPC allows Overmind to trace network-level misconfigurations that could affect traffic flow.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

When a regional internal forwarding rule is restricted to a particular subnetwork, the subnetwork is explicitly referenced. This link lets Overmind evaluate subnet-level controls (such as secondary ranges and IAM bindings) in the context of the forwarding rule’s traffic path.
