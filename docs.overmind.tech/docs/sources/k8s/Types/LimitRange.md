---
title: Limit Range
sidebar_label: LimitRange
---

A Kubernetes LimitRange is a namespace-level policy object that defines default, minimum, and maximum compute-resource constraints (such as CPU, memory, and ephemeral storage) that apply to Pods or individual Containers created in that namespace. By enforcing these boundaries, a LimitRange prevents a single workload from monopolising cluster resources and ensures that every workload has sensible defaults if the user omits explicit resource requests or limits. See the official Kubernetes documentation for full details: https://kubernetes.io/docs/concepts/policy/limit-range/

**Terrafrom Mappings:**

- `kubernetes_limit_range_v1.metadata[0].name`
- `kubernetes_limit_range.metadata[0].name`

## Supported Methods

- `GET`: Get a Limit Range by name
- `LIST`: List all Limit Ranges
- `SEARCH`: Search for a Limit Range using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
