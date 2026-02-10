---
title: Endpoints
sidebar_label: Endpoints
---

An Endpoint in Kubernetes represents the network locations (IP address + port) that actually implement a Service. While a Service is an abstract front-end, the corresponding Endpoints object keeps the ever-changing list of Pods that are ready to receive traffic. See the official Kubernetes documentation for full details: https://kubernetes.io/docs/concepts/services-networking/service/#endpoints

**Terrafrom Mappings:**

- `kubernetes_endpoints.metadata[0].name`
- `kubernetes_endpoints_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Endpoints by name
- `LIST`: List all Endpointss
- `SEARCH`: Search for a Endpoints using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Node`](/sources/k8s/Types/Node)

Each endpoint address can include a `nodeName` field indicating the Node on which the backing Pod is running. Overmind therefore links the Endpoints object to the Node(s) that currently host its backing Pods, helping you understand on which worker machines traffic will land.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Every endpoint entry exposes an IP address. Overmind extracts these IPs and links them, allowing you to trace the path from the abstract Service through the Endpoint to the concrete network address that will receive traffic.

### [`Pod`](/sources/k8s/Types/Pod)

Endpoint addresses typically contain a `targetRef` that points to the Pod providing the Service. Overmind links the Endpoints object to these Pods so you can quickly inspect the health, labels, and configuration of the workloads currently registered behind the Service.
