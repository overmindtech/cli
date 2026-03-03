---
title: GCP Big Table Admin Cluster
sidebar_label: gcp-big-table-admin-cluster
---

A GCP Bigtable Admin Cluster resource represents the configuration of a single cluster that belongs to a Cloud Bigtable instance. The cluster defines the geographic location where data is stored, the number and type of serving nodes, the storage type (HDD or SSD), autoscaling settings, and any customer-managed encryption keys (CMEK) that protect the data. It is managed through the Cloud Bigtable Admin API, which allows you to create, update, or delete clusters programmatically.  
For further details, see Google’s official documentation: https://cloud.google.com/bigtable/docs/instances-clusters-nodes

## Supported Methods

- `GET`: Get a gcp-big-table-admin-cluster by its "instances|clusters"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-big-table-admin-cluster by its "instances"

## Possible Links

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

A cluster is always a child of a Bigtable instance. This link represents the parent–child relationship: the instance contains one or more clusters, and every cluster must reference its parent instance.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled, the cluster’s encryption configuration points to the Cloud KMS CryptoKey that is used to encrypt data at rest. This link captures that dependency between the cluster and the key.
