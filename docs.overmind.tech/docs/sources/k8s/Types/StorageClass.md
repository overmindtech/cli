---
title: Storage Class
sidebar_label: StorageClass
---

A StorageClass is a cluster-wide Kubernetes resource that defines a “class” or tier of persistent storage that can be requested by workloads. Each StorageClass couples a provisioner (for example an AWS EBS driver, a CSI plug-in, or a Ceph back-end) with a set of parameters such as performance characteristics, encryption settings, reclaim policy, and mount options. When a user creates a PersistentVolumeClaim that references a particular `storageClassName`, Kubernetes dynamically provisions a matching PersistentVolume according to the rules in the StorageClass and binds it to the claim. This abstraction lets platform teams expose multiple quality-of-service levels while shielding application teams from underlying infrastructure details.  
Official documentation: https://kubernetes.io/docs/concepts/storage/storage-classes/

**Terrafrom Mappings:**

- `kubernetes_storage_class.metadata[0].name`
- `kubernetes_storage_class_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Storage Class by name
- `LIST`: List all Storage Classs
- `SEARCH`: Search for a Storage Class using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
