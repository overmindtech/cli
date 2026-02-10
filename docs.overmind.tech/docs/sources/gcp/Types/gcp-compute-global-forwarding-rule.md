---
title: GCP Compute Global Forwarding Rule
sidebar_label: gcp-compute-global-forwarding-rule
---

A Google Compute Engine **Global Forwarding Rule** represents the externally-visible IP address and port(s) that receive traffic for a global load balancer. It defines where packets that enter on a particular protocol/port combination should be sent, pointing them at a target proxy (for HTTP(S), SSL or TCP Proxy load balancers) or target VPN gateway. In the case of Internal Global Load Balancing it may also specify the VPC network and subnetwork that own the virtual IP address. In short, the forwarding rule is the public (or internal) entry-point that maps client traffic to the load balancer’s control plane.  
Official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/globalForwardingRules

**Terrafrom Mappings:**

- `google_compute_global_forwarding_rule.name`

## Supported Methods

- `GET`: Get a gcp-compute-global-forwarding-rule by its "name"
- `LIST`: List all gcp-compute-global-forwarding-rule
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

A global forwarding rule ultimately delivers traffic to one or more backend services via a chain of resources (target proxy → URL map → backend service). Overmind surfaces this indirect relationship so that you can trace the path from the exposed IP address all the way to the workloads that will handle the request.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When the forwarding rule is used for internal global load balancing, it contains a `network` field that points to the VPC network that owns the virtual IP address. This link allows Overmind to show which network the listener lives in and what other resources share that network.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Similar to the network link, internal forwarding rules may reference a specific `subnetwork`. Overmind records this connection so you can identify the exact IP range and region in which the internal load balancer’s virtual IP is allocated.
