---
title: GCP Monitoring Notification Channel
sidebar_label: gcp-monitoring-notification-channel
---

A Google Cloud Monitoring Notification Channel is a resource that specifies where and how alerting notifications are delivered from Cloud Monitoring. Channels can point to many target types – e-mail, SMS, mobile push, Slack, PagerDuty, Pub/Sub, webhooks and more – and each channel stores the parameters (addresses, tokens, templates, etc.) required to reach that destination. Alerting policies reference one or more notification channels so that, when a policy is triggered, Cloud Monitoring automatically sends the message to the configured recipients.  
For a full description of the resource and its schema, see the official Google Cloud documentation: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.notificationChannels.

**Terrafrom Mappings:**

- `google_monitoring_notification_channel.name`

## Supported Methods

- `GET`: Get a gcp-monitoring-notification-channel by its "name"
- `LIST`: List all gcp-monitoring-notification-channel
- `SEARCH`: Search by full resource name: projects/[project]/notificationChannels/[notificationChannel] (used for terraform mapping).
