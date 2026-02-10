---
title: GCP Spanner Database
sidebar_label: gcp-spanner-database
---

Google Cloud Spanner is Google Cloud’s fully-managed, horizontally-scalable, relational database service.  
A Spanner **database** is the logical container that holds your tables, schema objects and data inside a Spanner instance. Each database inherits the instance’s compute and storage configuration and can be encrypted either with Google-managed keys or with a customer-managed key (CMEK).  
For an overview of the service see the official documentation: https://cloud.google.com/spanner/docs/overview

**Terrafrom Mappings:**

- `google_spanner_database.name`

## Supported Methods

- `GET`: Get a gcp-spanner-database by its "instances|databases"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-spanner-database by its "instances"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Spanner database can be encrypted with a customer-managed encryption key (CMEK) stored in Cloud KMS. When CMEK is enabled, the database resource is linked to the specific `gcp-cloud-kms-crypto-key` that provides its encryption.

### [`gcp-spanner-instance`](/sources/gcp/Types/gcp-spanner-instance)

Every Spanner database lives inside a Spanner instance. The database inherits performance characteristics and regional configuration from its parent `gcp-spanner-instance`, making this a direct parent–child relationship.
