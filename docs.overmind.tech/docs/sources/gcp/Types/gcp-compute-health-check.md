---
title: GCP Compute Health Check
sidebar_label: gcp-compute-health-check
---

A GCP Compute Health Check is a Google Cloud resource that periodically probes virtual machine instances or endpoints to decide whether they are fit to receive production traffic. The check runs from the Google-managed control plane using protocols such as TCP, SSL, HTTP(S), HTTP/2 or gRPC, and compares the response to thresholds you configure (e.g. response code, timeout, healthy/unhealthy counts). Backend services, target pools and managed instance groups use the resulting health status to route requests only to healthy instances and to trigger autoscaling or fail-over behaviour. Health checks come in global and regional flavours, aligning with global and regional load balancers respectively.  
Official documentation: https://cloud.google.com/load-balancing/docs/health-checks

**Terrafrom Mappings:**

  * `google_compute_health_check.name`
  * `google_compute_region_health_check.name`

## Supported Methods

* `GET`: Get GCP Compute Health Check by "gcp-compute-health-check-name"
* `LIST`: List all GCP Compute Health Check items
* ~~`SEARCH`~~