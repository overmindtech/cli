---
title: GCP Pub Sub Topic
sidebar_label: gcp-pub-sub-topic
---

A Google Cloud Pub/Sub Topic is a named message stream into which publishers send messages and from which subscribers receive them. Topics act as the core distribution point in the Pub/Sub service, decoupling producers and consumers and enabling asynchronous, scalable communication between systems. For full details see the official documentation: https://docs.cloud.google.com/pubsub/docs/create-topic.

**Terrafrom Mappings:**

- `google_pubsub_topic.name`

## Supported Methods

- `GET`: Get a gcp-pub-sub-topic by its "name"
- `LIST`: List all gcp-pub-sub-topic
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Pub/Sub Topic can be encrypted with a customer-managed Cloud KMS key. When such a key is specified, the topic will hold a reference to the corresponding `gcp-cloud-kms-crypto-key`, and Overmind will surface this dependency so you can assess the impact of key rotation or removal.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets can be configured to send event notifications to a Pub/Sub Topic (for example, when objects are created or deleted). Overmind links the bucket to the topic so you can understand which storage resources rely on the topic and evaluate the blast radius of changes to either side.
