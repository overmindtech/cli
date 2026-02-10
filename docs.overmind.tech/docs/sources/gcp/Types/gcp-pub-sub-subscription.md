---
title: GCP Pub Sub Subscription
sidebar_label: gcp-pub-sub-subscription
---

A Google Cloud Pub/Sub subscription represents a named endpoint that receives messages that are published to a specific Pub/Sub topic. Subscribers pull (or are pushed) messages from the subscription, acknowledge them, and thereby remove them from the backlog. Subscriptions can be configured for pull or push delivery, control message-retention, enforce acknowledgement deadlines, use filtering, dead-letter topics or BigQuery/Cloud Storage sinks.  
For full details see the official documentation: https://cloud.google.com/pubsub/docs/subscriber

**Terrafrom Mappings:**

- `google_pubsub_subscription.name`

## Supported Methods

- `GET`: Get a gcp-pub-sub-subscription by its "name"
- `LIST`: List all gcp-pub-sub-subscription
- ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

A subscription can be of type “BigQuery subscription”, in which case Pub/Sub automatically streams all received messages into the linked BigQuery table. Overmind therefore links the subscription to the destination `gcp-big-query-table` so that you can see where your data will land.

### [`gcp-pub-sub-subscription`](/sources/gcp/Types/gcp-pub-sub-subscription)

Multiple subscriptions may exist on the same topic or share common dead-letter topics and filters. Overmind links related subscriptions together so you can understand fan-out patterns or duplicated consumption paths for the same data.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Every subscription is attached to exactly one topic, from which it receives messages. This parent–child relationship is surfaced by Overmind via a direct link to the source `gcp-pub-sub-topic`.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets can emit object-change notifications to Pub/Sub topics. Subscriptions that listen to such topics are therefore operationally coupled to the originating bucket. Overmind links the subscription to the relevant `gcp-storage-bucket` so you can trace the flow of change events from storage to message consumers.
