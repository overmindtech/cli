---
title: GCP File Instance
sidebar_label: gcp-file-instance
---

A GCP Filestore instance is a fully-managed network file system that provides high-performance, scalable Network File System (NFS) shares to Google Cloud workloads. It allows you to mount POSIX-compliant file storage from Compute Engine VMs, GKE clusters and other services without having to provision or manage the underlying storage infrastructure yourself. Each instance resides in a specific region and VPC network, exposes one or more IP addresses, and can be encrypted with either Google-managed or customer-managed keys.  
For full details, refer to the official documentation: https://cloud.google.com/filestore/docs.

**Terrafrom Mappings:**

- `google_filestore_instance.id`

## Supported Methods

- `GET`: Get a gcp-file-instance by its "locations|instances"
- ~~`LIST`~~
- `SEARCH`: Search for Filestore instances in a location. Use the location string or the full resource name supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Filestore instance can be configured to use a customer-managed encryption key (CMEK) stored in Cloud KMS. When CMEK is enabled, the instance has a direct dependency on the specified `gcp-cloud-kms-crypto-key`, and loss or revocation of that key will render the file share inaccessible.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every Filestore instance is attached to a single VPC network and is reachable through an internal IP address range that you specify. This link represents the network in which the instance’s NFS endpoints are published and through which client traffic must flow.
