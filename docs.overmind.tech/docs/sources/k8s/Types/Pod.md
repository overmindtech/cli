---
title: Pod
sidebar_label: Pod
---

A Kubernetes Pod is the smallest deployable unit in the Kubernetes object model. It represents one or more containers that share storage, network and a specification for how to run the containers. Pods are ephemeral and are usually created and managed by higher-level controllers such as Deployments or StatefulSets. See the official Kubernetes documentation for full details: https://kubernetes.io/docs/concepts/workloads/pods/

**Terrafrom Mappings:**

- `kubernetes_pod.metadata[0].name`
- `kubernetes_pod_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Pod by name
- `LIST`: List all Pods
- `SEARCH`: Search for a Pod using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`ConfigMap`](/sources/k8s/Types/ConfigMap)

Pods can consume ConfigMaps as environment variables or mount them as files, allowing configuration data to be injected without rebuilding container images.

### [`ec2-volume`](/sources/aws/Types/ec2-volume)

When a Pod mounts a PersistentVolume backed by an AWS Elastic Block Store (EBS) volume, that underlying storage appears here as an `ec2-volume` link, connecting the workload to the physical disk resource in AWS.

### [`dns`](/sources/stdlib/Types/dns)

Each Pod receives an internal DNS entry (`<pod-ip>.<namespace>.pod.cluster.local`) and may resolve or be resolved by other services; Overmind records this relationship so you can trace DNS dependencies.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

At runtime every Pod is assigned an IP address. This link surfaces the relationship between the Kubernetes object and the network IP resource managed by the underlying cloud networking layer.

### [`PersistentVolumeClaim`](/sources/k8s/Types/PersistentVolumeClaim)

Pods declare one or more PersistentVolumeClaims in their `volumes` section to obtain persistent storage. The link shows which claims are attached to the Pod.

### [`PriorityClass`](/sources/k8s/Types/PriorityClass)

A Pod may specify a `priorityClassName`; the associated PriorityClass influences scheduling order and pre-emption behaviour. This link ties the Pod to its scheduling priority.

### [`Secret`](/sources/k8s/Types/Secret)

Secrets can be mounted as files or injected as environment variables into a Pod, for example to provide credentials or TLS keys. This link identifies every Secret the Pod references.

### [`ServiceAccount`](/sources/k8s/Types/ServiceAccount)

Each Pod runs under a ServiceAccount that defines its Kubernetes API permissions and, in many cases, its cloud IAM identity. The link shows the ServiceAccount used by the Pod.
