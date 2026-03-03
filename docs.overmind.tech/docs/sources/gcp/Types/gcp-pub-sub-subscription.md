---
title: GCP Pub Sub Subscription
sidebar_label: gcp-pub-sub-subscription
---

A Google Cloud Pub/Sub subscription represents a stream of messages delivered from a single Pub/Sub topic to a consumer application. Each subscription defines how, where and for how long messages are retained, whether the delivery is push or pull, any filters or dead-letter policies, and the IAM principals that are allowed to read from it. Official documentation can be found at Google Cloud – Pub/Sub Subscriptions: https://cloud.google.com/pubsub/docs/subscription-overview

**Terrafrom Mappings:**

- `google_pubsub_subscription.name`
- `google_pubsub_subscription_iam_binding.subscription`
- `google_pubsub_subscription_iam_member.subscription`
- `google_pubsub_subscription_iam_policy.subscription`

## Supported Methods

- `GET`: Get a gcp-pub-sub-subscription by its "name"
- `LIST`: List all gcp-pub-sub-subscription
- ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Pub/Sub can deliver messages directly into BigQuery by means of a BigQuery subscription. When such an integration is configured, the subscription is linked to the destination BigQuery table.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Service accounts are granted roles such as `roles/pubsub.subscriber` on the subscription so that applications can pull or acknowledge messages, or so that Pub/Sub can impersonate them for push deliveries. These IAM bindings create a relationship between the subscription and the service accounts.

### [`gcp-pub-sub-subscription`](/sources/gcp/Types/gcp-pub-sub-subscription)

Multiple subscriptions can point at the same topic, or one subscription may forward undelivered messages to another subscription via a dead-letter topic. Overmind shows these peer or chained subscriptions as related items.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Every subscription is attached to exactly one topic. All messages published to that topic are made available to the subscription, making the topic the primary upstream dependency.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets can emit object-change notifications to a Pub/Sub topic. If the subscription listens to such a topic, it is indirectly linked to the bucket that generated the events, allowing you to trace the flow from storage changes to message consumption.
