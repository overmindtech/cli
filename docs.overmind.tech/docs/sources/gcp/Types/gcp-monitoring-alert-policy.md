---
title: GCP Monitoring Alert Policy
sidebar_label: gcp-monitoring-alert-policy
---

A GCP Monitoring Alert Policy defines the conditions under which Google Cloud Monitoring should raise an alert and the actions that should be taken when those conditions are met. It lets you specify metrics to watch, threshold values, duration, notification channels, documentation for responders, and incident autoclose behaviour. Alert policies are a core part of Google Cloud’s observability suite, helping operations teams detect and respond to issues before they affect end-users.  
For full details, see the official documentation: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies#AlertPolicy

**Terrafrom Mappings:**

- `google_monitoring_alert_policy.id`

## Supported Methods

- `GET`: Get a gcp-monitoring-alert-policy by its "name"
- `LIST`: List all gcp-monitoring-alert-policy
- `SEARCH`: Search by full resource name: projects/[project]/alertPolicies/[alert_policy_id] (used for terraform mapping).

## Possible Links

### [`gcp-monitoring-notification-channel`](/sources/gcp/Types/gcp-monitoring-notification-channel)

An alert policy can reference one or more Notification Channels. These channels determine where alerts are delivered (e-mail, SMS, Pub/Sub, PagerDuty, etc.). Overmind therefore creates a link from each gcp-monitoring-alert-policy to the gcp-monitoring-notification-channel resources it targets, allowing you to understand which channels will be invoked when a policy fires.
