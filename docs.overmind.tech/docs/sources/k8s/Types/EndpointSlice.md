---
title: Endpoint Slice
sidebar_label: EndpointSlice
---

EndpointSlices provide a scalable and extensible way of tracking network endpoints that back a Kubernetes Service. Each slice contains a list of IP addresses and ports together with optional topology information such as the Node on which each endpoint is running. EndpointSlices replace the legacy Endpoints object for large clusters and are automatically created and managed by the control plane when a Service is defined.  
For full details see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/

**Terrafrom Mappings:**

- `kubernetes_endpoints_slice_v1.metadata[0].name`
- `kubernetes_endpoints_slice.metadata[0].name`

## Supported Methods

- `GET`: Get a EndpointSlice by name
- `LIST`: List all EndpointSlices
- `SEARCH`: Search for a EndpointSlice using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Node`](/sources/k8s/Types/Node)

Each endpoint within an EndpointSlice may include a `nodeName` or topology label indicating the Node that hosts the backing Pod. Overmind links the slice to those Nodes so you can see which machines will receive traffic for the Service.

### [`Pod`](/sources/k8s/Types/Pod)

Endpoints usually correspond to Pod IPs. By linking EndpointSlices to the underlying Pods, Overmind allows you to trace from a Service to the exact workloads that will handle requests.

### [`dns`](/sources/stdlib/Types/dns)

When Kubernetes populates cluster DNS (e.g. `my-service.my-namespace.svc.cluster.local`) it ultimately resolves to the addresses listed in the Service’s EndpointSlices. Linking to DNS records shows how a name queried by applications maps to concrete endpoints.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

EndpointSlices store one or more IPv4/IPv6 addresses for each endpoint. These addresses are linked so that you can follow a path from a Service to the raw IPs that will be contacted, helping to assess network-level reachability and risk.
