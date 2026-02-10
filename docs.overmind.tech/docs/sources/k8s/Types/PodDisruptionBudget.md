---
title: Pod Disruption Budget
sidebar_label: PodDisruptionBudget
---

A PodDisruptionBudget (PDB) is a Kubernetes policy object that limits the number of pods of a replicated application that can be unavailable during voluntary disruptions such as a node drain, cluster upgrade, or a user-initiated rolling update. By defining either a `minAvailable` or `maxUnavailable` threshold, it helps you maintain a desired level of service availability while still allowing the platform to carry out maintenance tasks.  
See the official documentation for full details: https://kubernetes.io/docs/concepts/workloads/pods/disruptions/

**Terrafrom Mappings:**

- `kubernetes_pod_disruption_budget_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a PodDisruptionBudget by name
- `LIST`: List all PodDisruptionBudgets
- `SEARCH`: Search for a PodDisruptionBudget using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A PodDisruptionBudget references pods via a label selector defined in `spec.selector`. Any pod whose labels match this selector is governed by the PDB, meaning it counts towards the availability calculations and is protected from eviction when the defined disruption limits would be exceeded.
