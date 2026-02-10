---
title: Ingress
sidebar_label: Ingress
---

An Ingress is a Kubernetes resource that manages external access to services within a cluster, typically HTTP and HTTPS traffic. It defines a set of routing rules that map incoming requests (based on hostnames and URL paths) to backend `Service` resources. By centralising traffic management, it allows fine-grained control over features such as virtual hosting, TLS termination and path-based routing without requiring each service to expose its own `Service` of type `LoadBalancer` or `NodePort`.  
Official documentation: https://kubernetes.io/docs/concepts/services-networking/ingress/

**Terrafrom Mappings:**

- `kubernetes_ingress_v1.metadata[0].name`

## Supported Methods

- `GET`: Get an Ingress by name
- `LIST`: List all Ingresses
- `SEARCH`: Search for an Ingress using the `ListOptions` JSON format, e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Service`](/sources/k8s/Types/Service)

An Ingress routes external traffic to one or more backend `Service` objects. Each rule in the Ingress specification references a service name and port; therefore, Overmind links an Ingress to the `Service`(s) it targets so that you can trace how requests reach your application.

### [`dns`](/sources/stdlib/Types/dns)

The hostnames declared in an Ingress must be resolvable via DNS so that clients can reach the cluster’s ingress point. Overmind links these hostnames to their corresponding DNS records (A, AAAA or CNAME) to show whether the necessary records exist and to surface any misconfigurations.
