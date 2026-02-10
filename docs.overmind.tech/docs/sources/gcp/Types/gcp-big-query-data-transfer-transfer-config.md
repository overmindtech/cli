---
title: GCP Big Query Data Transfer Transfer Config
sidebar_label: gcp-big-query-data-transfer-transfer-config
---

The BigQuery Data Transfer Service Transfer Config defines a scheduled data-transfer job in Google Cloud. It specifies where the data comes from (for example Google Ads, YouTube or an external Cloud Storage bucket), the destination BigQuery dataset, the refresh window, schedule, run-options, encryption settings and notification preferences. In essence, it is the canonical object that tells BigQuery Data Transfer Service what to move, when to move it and how to handle the resulting tables.
Official documentation: https://docs.cloud.google.com/bigquery/docs/working-with-transfers

**Terrafrom Mappings:**

- `google_bigquery_data_transfer_config.id`

## Supported Methods

- `GET`: Get a gcp-big-query-data-transfer-transfer-config by its "locations|transferConfigs"
- ~~`LIST`~~
- `SEARCH`: Search for BigQuery Data Transfer transfer configs in a location. Use the format "location" or "projects/project_id/locations/location/transferConfigs/transfer_config_id" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

The transfer config’s `destinationDatasetId` points to the BigQuery dataset that will receive the imported data, so the config depends on – and is intrinsically linked to – that dataset.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption is enabled, the transfer config references a Cloud KMS CryptoKey that is used to encrypt the tables created by the transfer, creating a dependency on the key.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Through the `notificationPubsubTopic` field, the transfer config can publish status and error messages about individual transfer runs to a Pub/Sub topic, establishing an outgoing link to that topic.
