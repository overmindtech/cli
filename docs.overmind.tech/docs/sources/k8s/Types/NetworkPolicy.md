---
title: Network Policy
sidebar_label: NetworkPolicy
---

A Kubernetes **NetworkPolicy** is a namespaced resource that controls how groups of Pods are allowed to communicate with each other and with other network endpoints. By defining ingress and/or egress rules that match Pods via label selectors, it provides fine-grained, declarative network segmentation inside the cluster, helping operators restrict unintended traffic and harden workloads. If no NetworkPolicy targets a Pod, that Pod is non-isolated and can both send and receive traffic to and from any source.  
Official documentation: https://kubernetes.io/docs/concepts/services-networking/network-policies/

**Terrafrom Mappings:**

- `kubernetes_network_policy.metadata[0].name`
- `kubernetes_network_policy_v1.metadata[0].name`

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A NetworkPolicy selects one or more Pods (in the same namespace) through `podSelector` rules; therefore, each referenced Pod can be linked to the NetworkPolicy that governs its allowed ingress and egress traffic.
