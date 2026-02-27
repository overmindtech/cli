---
title: GCP Monitoring Notification Channel
sidebar_label: gcp-monitoring-notification-channel
---

A **Google Cloud Monitoring Notification Channel** specifies where and how Cloud Monitoring delivers alert notifications—for example via email, SMS, Cloud Pub/Sub, Slack or PagerDuty. Each channel stores the configuration necessary for a particular medium (address, webhook URL, Pub/Sub topic name, etc.) and can be referenced by one or more alerting policies. For full details, see the official Google documentation: https://cloud.google.com/monitoring/support/notification-options

**Terrafrom Mappings:**

* `google_monitoring_notification_channel.name`

## Supported Methods

* `GET`: Get a gcp-monitoring-notification-channel by its "name"
* `LIST`: List all gcp-monitoring-notification-channel
* `SEARCH`: Search by full resource name: projects/[project]/notificationChannels/[notificationChannel] (used for terraform mapping).

## Possible Links

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

If the notification channel’s `type` is `pubsub`, the channel references a specific Cloud Pub/Sub topic where alert messages are published. Overmind therefore links the notification channel to the corresponding `gcp-pub-sub-topic` resource so that you can trace how alerts propagate into event-driven workflows or downstream systems.
