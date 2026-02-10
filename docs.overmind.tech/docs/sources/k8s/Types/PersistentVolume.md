---
title: Persistent Volume
sidebar_label: PersistentVolume
---

A Kubernetes PersistentVolume (PV) is a cluster-wide object that represents a piece of storage that has been provisioned either statically by an administrator or dynamically via a StorageClass. Unlike ephemeral volumes that are tied to the lifetime of a Pod, a PV exists independently and can outlive any consumer Pods, enabling stateful workloads to retain data across rescheduling or restarts. Each PV encapsulates details such as capacity, access modes, reclaim policy and the specifics of the underlying storage medium (for example, AWS EBS, NFS, or a CSI-provisioned backend).  
Official documentation: https://kubernetes.io/docs/concepts/storage/persistent-volumes/

**Terrafrom Mappings:**

- `kubernetes_persistent_volume.metadata[0].name`
- `kubernetes_persistent_volume_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a PersistentVolume by name
- `LIST`: List all PersistentVolumes
- `SEARCH`: Search for a PersistentVolume using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`ec2-volume`](/sources/aws/Types/ec2-volume)

A PersistentVolume whose `spec.awsElasticBlockStore` (or CSI driver) references an AWS EBS disk ultimately maps to an EC2 volume. Overmind links the PV to the underlying `ec2-volume` so you can assess risks such as deletion protection, encryption status or capacity limits of the actual block device.

### [`efs-access-point`](/sources/aws/Types/efs-access-point)

When a PV is backed by Amazon EFS via the EFS CSI driver, it mounts the file system through a specific EFS Access Point. Linking the PV to the corresponding `efs-access-point` lets you trace permissions, throughput and network configurations that could affect the workload’s storage availability.

### [`StorageClass`](/sources/k8s/Types/StorageClass)

Most dynamically provisioned PVs include a `storageClassName` field that references the StorageClass used to create them. By linking to the `StorageClass`, Overmind shows the provisioning parameters, reclaim policy and allowed topologies that govern how this PV was created and how it behaves when released.
