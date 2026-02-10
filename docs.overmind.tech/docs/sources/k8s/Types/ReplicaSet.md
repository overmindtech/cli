---
title: Replica Set
sidebar_label: ReplicaSet
---

A ReplicaSet is a Kubernetes controller whose purpose is to maintain a stable set of identical Pods running at any given time. By continuously watching the cluster state, it ensures that the desired number of Pod replicas are present: if one is deleted or becomes unhealthy, the ReplicaSet will automatically create a replacement. ReplicaSets are most commonly created implicitly by Deployments, but they can also be defined directly.  
For full details, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/

## Supported Methods

- `GET`: Get a ReplicaSet by name
- `LIST`: List all ReplicaSets
- `SEARCH`: Search for a ReplicaSet using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A ReplicaSet owns and manages a collection of Pods that match its selector. Each linked Pod represents one replica maintained by the ReplicaSet; scaling or health-checking operations performed by the ReplicaSet directly affect these Pods.
