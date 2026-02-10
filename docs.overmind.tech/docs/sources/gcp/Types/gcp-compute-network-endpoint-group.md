---
title: GCP Compute Network Endpoint Group
sidebar_label: gcp-compute-network-endpoint-group
---

A Google Cloud Compute Network Endpoint Group (NEG) is a collection of network endpoints—VM NICs, IP and port pairs, or fully-managed serverless targets such as Cloud Run and Cloud Functions—that you treat as a single backend for Google Cloud Load Balancing. By grouping endpoints into a NEG you can precisely steer traffic, perform health-checking, and scale back-end capacity without exposing individual resources. See the official documentation for full details: https://cloud.google.com/load-balancing/docs/negs/.

**Terrafrom Mappings:**

- `google_compute_network_endpoint_group.name`

## Supported Methods

- `GET`: Get a gcp-compute-network-endpoint-group by its "name"
- `LIST`: List all gcp-compute-network-endpoint-group
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-functions-function`](/sources/gcp/Types/gcp-cloud-functions-function)

Serverless NEGs can reference a Cloud Functions function as their target, allowing the function to serve as a backend to an HTTP(S) load balancer. Overmind links a NEG to the Cloud Functions function it fronts.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A VM-based or hybrid NEG is created inside a specific VPC network; all its endpoints must belong to that network. Overmind therefore relates the NEG to the corresponding `gcp-compute-network`.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

For regional VM NEGs, each endpoint is an interface on a VM residing in a particular subnetwork. Overmind surfaces this dependency by linking the NEG to each associated `gcp-compute-subnetwork`.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

When a Cloud Run service is exposed through an external HTTP(S) load balancer, Google automatically creates a serverless NEG representing that service. Overmind links the NEG back to its originating `gcp-run-service`.
