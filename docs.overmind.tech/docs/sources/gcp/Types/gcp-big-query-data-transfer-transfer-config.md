---
title: GCP Big Query Data Transfer Transfer Config
sidebar_label: gcp-big-query-data-transfer-transfer-config
---

A BigQuery Data Transfer transfer configuration defines the schedule, destination dataset and credentials that the BigQuery Data Transfer Service will use to load data from a supported SaaS application, Google service or external data source into BigQuery. Each configuration specifies when transfers should run, the parameters required by the source system and, optionally, Pub/Sub notification settings and Cloud KMS encryption keys.  
For a full description of the resource see the Google Cloud documentation: https://cloud.google.com/bigquery/docs/reference/datatransfer/rest/v1/projects.locations.transferConfigs

**Terrafrom Mappings:**

- `google_bigquery_data_transfer_config.id`

## Supported Methods

- `GET`: Get a gcp-big-query-data-transfer-transfer-config by its "locations|transferConfigs"
- ~~`LIST`~~
- `SEARCH`: Search for BigQuery Data Transfer transfer configs in a location. Use the format "location" or "projects/project_id/locations/location/transferConfigs/transfer_config_id" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

The transfer configuration writes its imported data into a specific BigQuery dataset; the dataset’s identifier is stored in the configuration’s `destinationDatasetId` field. Overmind therefore links the config to the dataset that will receive the transferred data.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the destination dataset is protected with customer-managed encryption keys (CMEK), the transfer runs inherit that key. Consequently, the configuration is indirectly associated with the Cloud KMS crypto key that encrypts the loaded tables, allowing Overmind to surface encryption-related risks.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Transfers execute using a dedicated service account (`project-number@gcp-sa-bigquerydt.iam.gserviceaccount.com`) or, in some cases, a user-provided service account. The configuration stores this principal, and appropriate IAM roles must be granted. Overmind links the transfer config to the service account to assess permission scopes.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

A transfer configuration can be set to publish run status notifications to a Pub/Sub topic specified in its `notificationPubsubTopic` field. Overmind links the configuration to that topic so that message-flow and permissions between the two resources can be evaluated.
