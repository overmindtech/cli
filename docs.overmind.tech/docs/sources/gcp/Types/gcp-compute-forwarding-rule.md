---
title: GCP Compute Forwarding Rule
sidebar_label: gcp-compute-forwarding-rule
---

A GCP Compute Forwarding Rule defines how incoming packets are handled within Google Cloud. It binds an IP address, protocol and (optionally) port range to a specific target resource such as a backend service, target proxy or target pool. Forwarding rules underpin both external and internal load-balancing solutions and can be either regional or global in scope.  
For full details see the official documentation: https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts.

**Terrafrom Mappings:**

- `google_compute_forwarding_rule.name`

## Supported Methods

- `GET`: Get GCP Compute Forwarding Rule by "gcp-compute-forwarding-rule-name"
- `LIST`: List all GCP Compute Forwarding Rule items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

The forwarding rule may specify a backend service as its target (for example, when configuring an Internal TCP/UDP Load Balancer or External HTTP(S) Load Balancer).

### [`gcp-compute-forwarding-rule`](/sources/gcp/Types/gcp-compute-forwarding-rule)

This represents the same forwarding-rule resource; Overmind links to it so that self-references or associations between global and regional rules can be tracked.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

For internal forwarding rules, the rule is created inside a specific VPC network; the link identifies that parent network.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

If the rule’s IP address is allocated from a delegated public prefix, it will be linked to that prefix to show the allocation source.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Internal forwarding rules also reference the subnetwork from which their internal IP address is drawn.

### [`gcp-compute-target-http-proxy`](/sources/gcp/Types/gcp-compute-target-http-proxy)

External HTTP Load Balancer forwarding rules target an HTTP proxy, so the rule links to the relevant `target-http-proxy` resource.

### [`gcp-compute-target-https-proxy`](/sources/gcp/Types/gcp-compute-target-https-proxy)

External HTTPS Load Balancer forwarding rules target an HTTPS proxy; this link identifies that proxy.

### [`gcp-compute-target-pool`](/sources/gcp/Types/gcp-compute-target-pool)

Legacy Network Load Balancer forwarding rules can point directly to a target pool; the link shows which pool receives the traffic.
