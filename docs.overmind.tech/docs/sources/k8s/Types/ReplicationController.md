---
title: Replication Controller
sidebar_label: ReplicationController
---

A ReplicationController is a legacy Kubernetes workload controller whose job is to ensure that a specified number of pod replicas are running at any one time. If a pod crashes or is deleted, the ReplicationController creates a replacement; if too many exist, it deletes the excess. Although superseded by ReplicaSets and Deployments, ReplicationControllers are still respected by the Kubernetes API and may be encountered in older manifests. Further information can be found in the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller/

**Terrafrom Mappings:**

- `kubernetes_replication_controller.metadata[0].name`
- `kubernetes_replication_controller_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a ReplicationController by name
- `LIST`: List all ReplicationControllers
- `SEARCH`: Search for a ReplicationController using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A ReplicationController manages the lifecycle of a homogeneous set of Pods defined by its `spec.template`. Overmind links a ReplicationController to each Pod it owns via the `ownerReference`, enabling you to trace from controller to running workload (and vice-versa) when assessing deployment risk.
