---
title: GCP Compute Global Forwarding Rule
sidebar_label: gcp-compute-global-forwarding-rule
---

A Google Cloud Compute Global Forwarding Rule defines a single anycast virtual IP address that routes incoming traffic at the global level to a specified target (such as an HTTP(S) proxy, SSL proxy or TCP proxy) or, for internal load balancing, directly to a backend service. It is the entry-point resource for most external HTTP(S) and proxy load balancers and for internal global load balancers. For full details see the Google Cloud documentation: https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts

**Terrafrom Mappings:**

  * `google_compute_global_forwarding_rule.name`

## Supported Methods

* `GET`: Get a gcp-compute-global-forwarding-rule by its "name"
* `LIST`: List all gcp-compute-global-forwarding-rule
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

When the forwarding rule is created for an internal global load balancer, it references a backend service directly; the rule’s traffic is delivered to the backends listed in that service. Analysing this link lets Overmind trace traffic paths from the VIP to the actual instances or endpoints.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Internal global forwarding rules must be attached to a specific VPC network. Linking to the network resource reveals which project-wide connectivity domain the VIP belongs to and helps surface risks such as unintended exposure to peered networks.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

If the forwarding rule is internal, it is scoped to a particular subnetwork. Understanding this relationship identifies the IP range in which the virtual IP lives and highlights segmentation or overlapping-CIDR issues.

### [`gcp-compute-target-http-proxy`](/sources/gcp/Types/gcp-compute-target-http-proxy)

For external HTTP(S), SSL or TCP proxy load balancers, the forwarding rule points to a target proxy resource. The proxy terminates the client connection before forwarding to backend services. Linking these resources enables Overmind to trace configuration chains and detect misconfigurations such as SSL policy mismatches or missing backends.