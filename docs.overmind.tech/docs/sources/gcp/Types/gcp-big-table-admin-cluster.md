---
title: GCP Big Table Admin Cluster
sidebar_label: gcp-big-table-admin-cluster
---

A Cloud Bigtable cluster represents the set of serving and storage resources that handle all reads and writes for a Cloud Bigtable instance. Each cluster belongs to a single instance, lives in one Google Cloud zone, and is configured with a certain number of nodes and a specific storage type (SSD or HDD). Clusters can be added or removed to provide high availability, geographic redundancy, or additional throughput. With Overmind you can surface mis-configurations such as a single-zone deployment, inadequate node counts, or missing encryption settings before your change reaches production.  
Official Google documentation: https://cloud.google.com/bigtable/docs/overview#clusters

## Supported Methods

- `GET`: Get a gcp-big-table-admin-cluster by its "instances|clusters"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-big-table-admin-cluster by its "instances"

## Possible Links

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

Every cluster is a child resource of a Cloud Bigtable instance. Overmind links the cluster back to its parent instance so you can see which database workloads will be affected if you modify or delete the cluster.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

When customer-managed encryption keys (CMEK) are enabled for a Bigtable cluster, the cluster references a Cloud KMS crypto key. Overmind creates a link to that key so you can verify the key’s status, rotation schedule, and IAM policy before deploying changes to the cluster.
