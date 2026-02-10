---
title: Stateful Set
sidebar_label: StatefulSet
---

A StatefulSet is a Kubernetes workload controller that manages the deployment and scaling of a set of Pods, while guaranteeing the ordering and uniqueness of those Pods. Unlike Deployments, which are optimised for stateless services, StatefulSets are designed for applications that require stable network identities, stable persistent storage and ordered, graceful deployment and scaling. Typical use-cases include databases, distributed filesystems and clustered applications where each replica must be uniquely addressable.  
For full details, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/

**Terrafrom Mappings:**

- `kubernetes_stateful_set_v1.metadata[0].name`
- `kubernetes_stateful_set.metadata[0].name`

## Supported Methods

- `GET`: Get a Stateful Set by name
- `LIST`: List all Stateful Sets
- `SEARCH`: Search for a Stateful Set using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
