---
title: GCP Compute Network Endpoint Group
sidebar_label: gcp-compute-network-endpoint-group
---

A Google Cloud Platform Compute Network Endpoint Group (NEG) is a collection of network endpoints—such as VM NICs, container pods, Cloud Run services, or Cloud Functions—that can be treated as a single backend target by Load Balancing and Service Directory. NEGs give fine-grained control over which exact endpoints receive traffic and allow serverless or hybrid back-ends to participate in layer-4/7 load balancing. See the official documentation for full details: https://cloud.google.com/load-balancing/docs/negs.

**Terrafrom Mappings:**

  * `google_compute_network_endpoint_group.name`

## Supported Methods

* `GET`: Get a gcp-compute-network-endpoint-group by its "name"
* `LIST`: List all gcp-compute-network-endpoint-group
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-functions-function`](/sources/gcp/Types/gcp-cloud-functions-function)

A serverless NEG can reference a specific Cloud Function. Overmind therefore links the NEG to the underlying `gcp-cloud-functions-function` it represents, showing which function will receive traffic through the load balancer.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Zonal and regional NEGs are created inside a particular VPC network. The link indicates the network context in which the endpoints exist, helping to surface routing and firewall considerations.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

When a NEG is scoped to a subnetwork (for example for VM or GKE pod endpoints), Overmind links it to that subnetwork so you can trace how traffic enters specific IP ranges.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

Serverless NEGs can point to Cloud Run services. This link shows which `gcp-run-service` is exposed through the NEG and subsequently through any HTTP(S) load balancer.