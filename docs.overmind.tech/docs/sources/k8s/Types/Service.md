---
title: Service
sidebar_label: Service
---

A Kubernetes Service is an abstract resource that defines a logical set of Pods and the policy by which they can be accessed. It provides a stable virtual IP (ClusterIP), DNS entry and, depending on the type, can expose workloads internally within the cluster or externally to the Internet through NodePorts or cloud load-balancers. Services decouple network identity and discovery from the underlying Pods, allowing them to scale up, down, or be replaced without changing the connection endpoint.  
For full details see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/services-networking/service/

**Terrafrom Mappings:**

- `kubernetes_service.metadata[0].name`
- `kubernetes_service_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Service by name
- `LIST`: List all Services
- `SEARCH`: Search for a Service using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A Service selects one or more Pods via label selectors and forwards traffic to them. Overmind links Services to the Pods that currently match their selector so you can see which workloads will receive traffic.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Each Service is assigned one or more IP addresses (ClusterIP, ExternalIP, LoadBalancer IP). Overmind creates links to these IP resources to show the concrete network endpoints associated with the Service.

### [`dns`](/sources/stdlib/Types/dns)

Kubernetes automatically registers DNS records for every Service (e.g., `my-service.my-namespace.svc.cluster.local`). Overmind links Services to their corresponding DNS entries so you can trace name resolution to the backing workloads.

### [`Endpoints`](/sources/k8s/Types/Endpoints)

Each Service creates a corresponding Endpoints object with the same name that lists the IP addresses of the backing Pods. Overmind links Services to their Endpoints so you can see which addresses are currently active. This uses the legacy `core/v1` API and works on all Kubernetes versions.

### [`EndpointSlice`](/sources/k8s/Types/EndpointSlice)

Modern Kubernetes clusters create EndpointSlices (labelled with `kubernetes.io/service-name`) as the scalable replacement for Endpoints. Overmind searches for EndpointSlices matching the Service name so you can trace from a Service to the network endpoints that back it on newer clusters.
