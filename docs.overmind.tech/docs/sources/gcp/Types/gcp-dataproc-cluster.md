---
title: GCP Dataproc Cluster
sidebar_label: gcp-dataproc-cluster
---

A Google Cloud Dataproc Cluster is a managed cluster of Compute Engine virtual machines that runs open-source data-processing frameworks such as Apache Spark, Apache Hadoop, Presto and Trino. Dataproc handles the provisioning, configuration and ongoing management of the cluster, allowing you to submit jobs or create ephemeral clusters on demand while paying only for the compute you use. For full feature details see the official documentation: https://docs.cloud.google.com/dataproc/docs/concepts/overview.

**Terrafrom Mappings:**

- `google_dataproc_cluster.name`

## Supported Methods

- `GET`: Get a gcp-dataproc-cluster by its "name"
- `LIST`: List all gcp-dataproc-cluster
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Dataproc cluster can be configured to use a customer-managed encryption key (CMEK) from Cloud KMS to encrypt the persistent disks attached to its nodes as well as the cluster’s Cloud Storage staging bucket.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

Each Dataproc cluster is built from a specific Dataproc image (e.g., `2.1-debian11`). The image determines the operating system and the versions of Hadoop, Spark and other components installed on the VM instances.

### [`gcp-compute-instance-group-manager`](/sources/gcp/Types/gcp-compute-instance-group-manager)

Behind the scenes Dataproc creates managed instance groups for the primary, secondary and optional pre-emptible worker node pools. These MIGs handle instance creation, health-checking and replacement.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

The cluster’s VMs are attached to a specific VPC network, determining their routability and ability to reach other Google Cloud services or on-premises systems.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If you run Dataproc on sole-tenant nodes, the cluster associates each VM with a Compute Node Group to guarantee dedicated physical hardware.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within the chosen VPC, the cluster can be pinned to a particular subnetwork to control IP address ranges, firewall rules and routing.

### [`gcp-dataproc-autoscaling-policy`](/sources/gcp/Types/gcp-dataproc-autoscaling-policy)

Clusters may reference an Autoscaling Policy that automatically adds or removes worker nodes based on YARN or Spark metrics, optimising performance and cost.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every Dataproc node runs under a Compute Engine service account. This account’s IAM roles determine the cluster’s permission to read/write Cloud Storage, publish metrics, access BigQuery, etc.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Dataproc uses Cloud Storage buckets for staging job files, storing cluster logs and optionally as a default HDFS replacement via the `gcs://` connector. The cluster therefore references one or more buckets during its lifecycle.
