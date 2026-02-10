---
title: Priority Class
sidebar_label: PriorityClass
---

A Kubernetes `PriorityClass` is a cluster-wide, non-namespaced resource that defines the relative importance of Pods. Each PriorityClass carries an integer value; the higher the value, the earlier the scheduler will try to place Pods that reference it. PriorityClasses are also used during pre-emption: when the cluster is under resource pressure, Pods with lower priority may be evicted in favour of higher-priority Pods. For full details, refer to the official Kubernetes documentation: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass

**Terrafrom Mappings:**

- `kubernetes_priority_class_v1.metadata[0].name`
- `kubernetes_priority_class.metadata[0].name`

## Supported Methods

- `GET`: Get a Priority Class by name
- `LIST`: List all Priority Classs
- `SEARCH`: Search for a Priority Class using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
