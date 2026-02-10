---
title: Horizontal Pod Autoscaler
sidebar_label: HorizontalPodAutoscaler
---

The Horizontal Pod Autoscaler (HPA) is a native Kubernetes controller that automatically increases or decreases the number of running Pods in a Deployment, ReplicaSet, StatefulSet, or other scalable resource so that observed resource consumption stays close to a user-defined target. It polls the Kubernetes Metrics Server (or a custom/external metrics API) at a regular interval, compares CPU, memory, or arbitrary custom metrics against the specified thresholds, and then adjusts the `spec.replicas` field of the target workload accordingly. This enables applications to meet fluctuating demand without manual intervention or unnecessary over-provisioning, while still preventing sudden traffic spikes from overwhelming the cluster. You can read the full upstream specification in the official Kubernetes documentation: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/.

**Terrafrom Mappings:**

- `kubernetes_horizontal_pod_autoscaler_v2.metadata[0].name`

## Supported Methods

- `GET`: Get a Horizontal Pod Autoscaler by name
- `LIST`: List all Horizontal Pod Autoscalers
- `SEARCH`: Search for a Horizontal Pod Autoscaler using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
