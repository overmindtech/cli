---
title: GCP Big Table Admin Instance
sidebar_label: gcp-big-table-admin-instance
---

Google Cloud Bigtable is Google’s fully managed, scalable NoSQL database service.  
A Bigtable _instance_ is the administrative parent resource that defines the geographic placement, replication strategy, encryption settings and service-level configuration for the tables that will live inside it. Every instance contains one or more clusters, and each cluster in turn contains the nodes that serve user data. Creating or modifying an instance therefore determines where and how your Bigtable data will be stored and replicated.  
For further details, refer to the official Google Cloud documentation: https://cloud.google.com/bigtable/docs/instances-clusters-nodes

**Terrafrom Mappings:**

- `google_bigtable_instance.name`

## Supported Methods

- `GET`: Get a gcp-big-table-admin-instance by its "name"
- `LIST`: List all gcp-big-table-admin-instance
- ~~`SEARCH`~~

## Possible Links

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

A Bigtable Admin Instance is the parent of one or more Bigtable Admin Clusters. Each cluster resource belongs to exactly one instance, inheriting its replication and localisation settings. When Overmind discovers or updates a gcp-big-table-admin-instance, it follows this relationship to enumerate the gcp-big-table-admin-cluster resources that compose the instance’s underlying serving infrastructure.
