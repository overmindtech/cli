---
title: GCP Pub Sub Topic
sidebar_label: gcp-pub-sub-topic
---

A **Cloud Pub/Sub Topic** is a named message channel in Google Cloud Platform that receives messages from publishers and delivers them to subscribers. Topics decouple senders and receivers, allowing highly-scalable, asynchronous communication between services. Every message published to a topic is retained for the duration of its acknowledgement window and can be encrypted with a customer-managed key.  
For comprehensive information, see the official documentation: https://cloud.google.com/pubsub/docs/create-topic.

**Terrafrom Mappings:**

* `google_pubsub_topic.name`
* `google_pubsub_topic_iam_binding.topic`
* `google_pubsub_topic_iam_member.topic`
* `google_pubsub_topic_iam_policy.topic`

## Supported Methods

* `GET`: Get a gcp-pub-sub-topic by its "name"
* `LIST`: List all gcp-pub-sub-topic
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Pub/Sub topic may be encrypted using a customer-managed encryption key (CMEK). When CMEK is enabled, the topic resource holds a reference to the Cloud KMS Crypto Key that protects message data at rest.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Access to publish or subscribe is controlled through IAM roles that are granted to service accounts on the topic. The topic’s IAM policy therefore links it to any service account that has roles such as `roles/pubsub.publisher` or `roles/pubsub.subscriber` on the resource.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets can be configured to send change notifications to a Pub/Sub topic (for example, object create or delete events). In such configurations, the bucket acts as a publisher, and the topic appears as a dependent destination for bucket event notifications.
