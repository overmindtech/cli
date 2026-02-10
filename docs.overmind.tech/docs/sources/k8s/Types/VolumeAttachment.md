---
title: Volume Attachment
sidebar_label: VolumeAttachment
---

A Kubernetes `VolumeAttachment` represents the intent to attach (or detach) a PersistentVolume to a specific Node. It is created and managed automatically by the external CSI attacher or the in-tree volume controller whenever a Pod that uses a PersistentVolume is scheduled. Kubernetes will not make the volume available to the Pod until the corresponding `VolumeAttachment` reports that the attach operation has completed successfully.  
For full details see the official Kubernetes documentation: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume-attachment-v1/

## Supported Methods

- `GET`: Get a VolumeAttachment by name
- `LIST`: List all VolumeAttachments
- `SEARCH`: Search for a VolumeAttachment using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`PersistentVolume`](/sources/k8s/Types/PersistentVolume)

`VolumeAttachment.spec.source.persistentVolumeName` holds the name of the PersistentVolume to be attached. Overmind links the `VolumeAttachment` to this `PersistentVolume` so you can trace which physical storage device is being mounted on which node.

### [`Node`](/sources/k8s/Types/Node)

`VolumeAttachment.spec.nodeName` identifies the Node where the volume should be attached. Linking `VolumeAttachment` to the `Node` lets you understand which worker machine will host the volume and helps assess the impact of node-specific storage operations.
