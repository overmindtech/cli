---
title: GCP Compute Backend Service
sidebar_label: gcp-compute-backend-service
---

A Compute Backend Service defines how Google Cloud Load Balancers distribute traffic to one or more back-end targets (Instance Groups, Network Endpoint Groups, or serverless workloads). It specifies the load-balancing algorithm, session affinity, capacity controls, health checks, time-outs, protocol and (optionally) a Cloud Armor security policy. Backend services exist as either regional or global resources, depending on the load balancer type.  
For full details see the official Google Cloud documentation: https://cloud.google.com/load-balancing/docs/backend-service

**Terrafrom Mappings:**

- `google_compute_backend_service.name`
- `google_compute_region_backend_service.name`

## Supported Methods

- `GET`: Get GCP Compute Backend Service by "gcp-compute-backend-service-name"
- `LIST`: List all GCP Compute Backend Service items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-health-check`](/sources/gcp/Types/gcp-compute-health-check)

A backend service is required to reference one or more Health Checks. These determine the health of each backend target and whether traffic should be sent to it.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Individual VM instances receive traffic indirectly through a backend service when they belong to an instance group or unmanaged instance list that the backend service uses.

### [`gcp-compute-instance-group`](/sources/gcp/Types/gcp-compute-instance-group)

Managed or unmanaged Instance Groups are the most common type of backend that a backend service points to. The group’s VMs are the actual targets for load-balanced traffic.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Backends referenced by a backend service must reside in a specific VPC network; therefore the backend service is effectively bound to that network and its associated subnets and firewall rules.

### [`gcp-compute-network-endpoint-group`](/sources/gcp/Types/gcp-compute-network-endpoint-group)

Network Endpoint Groups (NEGs) can be configured as backends of a backend service to route traffic to endpoints such as containers, serverless services, or on-premises resources.

### [`gcp-compute-security-policy`](/sources/gcp/Types/gcp-compute-security-policy)

A backend service can optionally attach a Cloud Armor Security Policy to enforce L7 firewall rules, rate limiting, and other protective measures on incoming traffic before it reaches the backends.
