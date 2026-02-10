---
title: Persistent Volume Claim
sidebar_label: PersistentVolumeClaim
---

A PersistentVolumeClaim (PVC) in Kubernetes is a user-defined request for storage. Applications declare the amount of space, access mode and other requirements they need through a PVC, and Kubernetes finds (or waits for) a matching PersistentVolume (PV) to satisfy that request. Once bound, the PVC provides a stable, pod-agnostic handle for the underlying storage, meaning workloads can be rescheduled across nodes without losing data.  
For a full explanation see the Kubernetes documentation: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims

**Terrafrom Mappings:**

- `kubernetes_persistent_volume_claim.metadata[0].name`
- `kubernetes_persistent_volume_claim_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a PersistentVolumeClaim by name
- `LIST`: List all PersistentVolumeClaims
- `SEARCH`: Search for a PersistentVolumeClaim using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`PersistentVolume`](/sources/k8s/Types/PersistentVolume)

A PVC is bound to a PersistentVolume that satisfies its storage class, capacity and access-mode requirements. Overmind records this binding so that from a PVC you can quickly navigate to the backing PV and assess its characteristics and any associated risks.
