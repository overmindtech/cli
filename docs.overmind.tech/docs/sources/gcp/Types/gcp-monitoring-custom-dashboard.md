---
title: GCP Monitoring Custom Dashboard
sidebar_label: gcp-monitoring-custom-dashboard
---

A GCP Monitoring Custom Dashboard is a user-defined collection of charts and widgets that presents metrics, logs, and alerts for resources running in Google Cloud or on-premises. It allows platform teams to visualise performance, capacity, and health in a single view that can be shared across projects. Custom dashboards are managed through Cloud Monitoring and can be created or modified via the Google Cloud Console, the Cloud Monitoring API, or infrastructure-as-code tools such as Terraform.  
For full details, see the official documentation: https://cloud.google.com/monitoring/charts/dashboards

**Terrafrom Mappings:**

* `google_monitoring_dashboard.id`

## Supported Methods

* `GET`: Get a gcp-monitoring-custom-dashboard by its "name"
* `LIST`: List all gcp-monitoring-custom-dashboard
* `SEARCH`: Search for custom dashboards by their ID in the form of "projects/[project_id]/dashboards/[dashboard_id]". This is supported for terraform mappings.
