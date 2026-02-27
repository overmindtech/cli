---
title: GCP Monitoring Alert Policy
sidebar_label: gcp-monitoring-alert-policy
---

A Google Cloud Monitoring Alert Policy is a configuration object that defines the conditions under which Cloud Monitoring should create an incident, how incidents are grouped, and which notification channels should be used to inform operators. Alert policies enable proactive observation of metrics, logs and uptime checks across Google Cloud services so that you can respond quickly to anomalies. For more detail see the official Google Cloud documentation: [Create and manage alerting policies](https://cloud.google.com/monitoring/alerts).

**Terrafrom Mappings:**

* `google_monitoring_alert_policy.id`

## Supported Methods

* `GET`: Get a gcp-monitoring-alert-policy by its "name"
* `LIST`: List all gcp-monitoring-alert-policy
* `SEARCH`: Search by full resource name: projects/[project]/alertPolicies/[alert_policy_id] (used for terraform mapping).

## Possible Links

### [`gcp-monitoring-notification-channel`](/sources/gcp/Types/gcp-monitoring-notification-channel)

An alert policy can reference one or more notification channels so that, when its conditions are met, Cloud Monitoring sends notifications (e-mails, webhooks, SMS, etc.) through the linked gcp-monitoring-notification-channels.
