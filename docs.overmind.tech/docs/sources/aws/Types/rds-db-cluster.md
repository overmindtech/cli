---
title: RDS Cluster
sidebar_label: rds-db-cluster
---

Amazon Relational Database Service (RDS) Clusters provide a managed, highly-available relational database running on multiple Availability Zones. An RDS Cluster contains one or more database instances that share storage, backups, and endpoints, and can be configured for automatic fail-over and read-scaling. Aurora MySQL and Aurora PostgreSQL engines run exclusively within clusters, while other engines (e.g. MySQL, PostgreSQL) can participate in global database topologies through cluster links.  
Official documentation: https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/CHAP_AuroraOverview.html

**Terrafrom Mappings:**

- `aws_rds_cluster.cluster_identifier`

## Supported Methods

- `GET`: Get a cluster by identifier
- `LIST`: List all RDS clusters
- `SEARCH`: Search for a cluster by ARN

## Possible Links

### [`rds-db-subnet-group`](/sources/aws/Types/rds-db-subnet-group)

Each RDS Cluster is associated with a DB subnet group that defines the set of subnets (and therefore Availability Zones) in which its instances can run.

### [`dns`](/sources/stdlib/Types/dns)

The cluster exposes an endpoint such as `mycluster.cluster-123456789012.eu-west-2.rds.amazonaws.com`; this hostname is represented as a DNS record linked to the cluster.

### [`rds-db-cluster`](/sources/aws/Types/rds-db-cluster)

Clusters can reference other clusters as replication sources or targets (e.g. in an Aurora global database), creating a dependency link between the participating RDS clusters.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Traffic to and from the cluster’s instances is controlled by one or more EC2 security groups attached to the cluster.

### [`route53-hosted-zone`](/sources/aws/Types/route53-hosted-zone)

Organisations often create Route 53 records (A/AAAA or CNAME) in their hosted zones to provide friendly names for the cluster endpoint, linking the hosted zone to the RDS Cluster.

### [`kms-key`](/sources/aws/Types/kms-key)

If storage encryption is enabled, the cluster uses a customer-managed or AWS-managed KMS key; compromising or deleting the key will render the data inaccessible.

### [`rds-option-group`](/sources/aws/Types/rds-option-group)

Certain engines allow additional features to be enabled via option groups (e.g. Oracle options); a cluster may reference an option group to configure those extensions.

### [`iam-role`](/sources/aws/Types/iam-role)

An RDS Cluster can assume IAM roles for tasks such as exporting snapshots to S3, publishing logs to CloudWatch, or accessing AWS services like Kinesis; these roles are linked resources.
