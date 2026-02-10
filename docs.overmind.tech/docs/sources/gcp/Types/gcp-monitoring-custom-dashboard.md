---
title: GCP Monitoring Custom Dashboard
sidebar_label: gcp-monitoring-custom-dashboard
---

A Google Cloud Monitoring Custom Dashboard is a user-defined workspace in which you can visualise metrics, logs-based metrics and alerting information collected from your Google Cloud resources and external services. By assembling charts, heatmaps, and scorecards that matter to your organisation, a custom dashboard lets you observe the real-time health and historical behaviour of your workloads, share operational insights with your team, and troubleshoot incidents more quickly. Dashboards are created and managed through the Cloud Monitoring API, the Google Cloud console, or declaratively via Terraform.  
Official documentation: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#Dashboard

**Terrafrom Mappings:**

- `google_monitoring_dashboard.id`

## Supported Methods

- `GET`: Get a gcp-monitoring-custom-dashboard by its "name"
- `LIST`: List all gcp-monitoring-custom-dashboard
- `SEARCH`: Search for custom dashboards by their ID in the form of "projects/[project_id]/dashboards/[dashboard_id]". This is supported for terraform mappings.
