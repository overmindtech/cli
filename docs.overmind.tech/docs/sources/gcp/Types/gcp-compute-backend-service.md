---
title: GCP Compute Backend Service
sidebar_label: gcp-compute-backend-service
---

A GCP Compute Backend Service is the central configuration object that tells a Google Cloud load balancer where and how to send traffic.  
It groups one or more back-end targets (for example instance groups, zonal NEG or serverless NEG), specifies the load-balancing scheme (internal or external), session affinity, health checks, protocol, timeout and (optionally) Cloud Armor security policies.  
Because almost every Google Cloud load-balancing product routes traffic through a backend service, it is a critical part of any production deployment.  
Official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/backendServices

**Terrafrom Mappings:**

- `google_compute_backend_service.name`

## Supported Methods

- `GET`: Get GCP Compute Backend Service by "gcp-compute-backend-service-name"
- `LIST`: List all GCP Compute Backend Service items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A backend service implicitly belongs to the same VPC network as the back-end resources (instance groups or NEGs) it references. Consequently, the service’s reachability, IP ranges and firewall posture are constrained by that network, so Overmind creates a link to the corresponding `gcp-compute-network` to surface these dependencies.

### [`gcp-compute-security-policy`](/sources/gcp/Types/gcp-compute-security-policy)

If Cloud Armor is enabled, the backend service contains a direct reference to a `securityPolicy`. This link allows Overmind to show how web-application-firewall rules and rate-limiting policies are applied to traffic flowing through the backend service.
