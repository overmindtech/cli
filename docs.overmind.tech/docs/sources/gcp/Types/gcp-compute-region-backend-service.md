---
title: GCP Compute Region Backend Service
sidebar_label: gcp-compute-region-backend-service
---

A **GCP Compute Region Backend Service** is a regional load-balancing resource that defines how traffic is distributed to one or more back-end targets (such as Managed Instance Groups or Network Endpoint Groups) that all live in the same Google Cloud region. The service specifies settings such as the load-balancing protocol (HTTP, HTTPS, TCP, SSL etc.), session affinity, connection draining, health checks, fail-over behaviour and (optionally) Cloud Armor security policies. Regional backend services are used by Internal HTTP(S) Load Balancers, Internal TCP/UDP Load Balancers and several other Google Cloud load-balancing products.  
Official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/regionBackendServices

**Terrafrom Mappings:**

- `google_compute_region_backend_service.name`

## Supported Methods

- `GET`: Get GCP Compute Region Backend Service by "gcp-compute-region-backend-service-name"
- `LIST`: List all GCP Compute Region Backend Service items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-instance-group`](/sources/gcp/Types/gcp-compute-instance-group)

A region backend service lists one or more Managed Instance Groups (or unmanaged instance groups) as its back-ends; the load balancer distributes traffic across the VMs contained in these instance groups.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

For internal load balancing, the region backend service is tied to a specific VPC network. All back-ends must reside in subnets that belong to this network and traffic from the forwarding rule is delivered through it.

### [`gcp-compute-security-policy`](/sources/gcp/Types/gcp-compute-security-policy)

A backend service can optionally reference a Cloud Armor security policy. When attached, that policy governs and filters incoming requests before they reach the back-end targets.
