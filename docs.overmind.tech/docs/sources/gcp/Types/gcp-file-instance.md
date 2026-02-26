---
title: GCP File Instance
sidebar_label: gcp-file-instance
---

A GCP File Instance represents a Cloud Filestore instance – a managed network file storage appliance that provides an NFSv3 or NFSv4-compatible file share, typically used by GKE clusters or Compute Engine VMs that require shared, POSIX-compliant storage. Each instance is created in a specific GCP region and zone, connected to a VPC network, and exposes one or more file shares (called “filesets”) over a private RFC-1918 address. Instances can be customised for capacity and performance tiers, and may optionally use customer-managed encryption keys (CMEK) for data-at-rest encryption.  
Official documentation: https://cloud.google.com/filestore/docs/overview

**Terrafrom Mappings:**

  * `google_filestore_instance.id`

## Supported Methods

* `GET`: Get a gcp-file-instance by its "locations|instances"
* ~~`LIST`~~
* `SEARCH`: Search for Filestore instances in a location. Use the location string or the full resource name supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Filestore instance can be encrypted with a customer-managed Cloud KMS key (CMEK). The link shows which KMS Crypto Key is protecting the data-at-rest of this storage appliance.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Filestore instances are deployed into and reachable through a specific VPC network. This link identifies the Compute Network whose subnet provides the private IP addresses through which clients access the file share.