---
title: Daemon Set
sidebar_label: DaemonSet
---

A Kubernetes **DaemonSet** ensures that a copy of a specified Pod is running on every (or a selected subset of) node(s) in the cluster. It is commonly used for cluster-wide services such as log collectors, monitoring agents, or network proxies that must be present on each node. When nodes are added to the cluster, the DaemonSet automatically schedules the Pod on the new nodes; when nodes are removed, the Pods are garbage-collected.  
For a full description, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/

**Terrafrom Mappings:**

- `kubernetes_daemon_set_v1.metadata[0].name`
- `kubernetes_daemonset.metadata[0].name`

## Supported Methods

- `GET`: Get a Daemon Set by name
- `LIST`: List all Daemon Sets
- `SEARCH`: Search for a Daemon Set using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
