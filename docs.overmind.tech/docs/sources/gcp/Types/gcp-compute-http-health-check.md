---
title: GCP Compute Http Health Check
sidebar_label: gcp-compute-http-health-check
---

A GCP Compute HTTP Health Check is a globally scoped resource that periodically sends HTTP requests to a specified port and path on your instances or endpoints to verify that they are responding correctly. Load balancers, managed instance groups and other Google Cloud services use the results of these checks to decide whether traffic should be routed to a given backend. Each check can be customised with parameters such as the request path, host header, check interval, timeout, and healthy/unhealthy thresholds.  
For further details see the official documentation: https://cloud.google.com/compute/docs/load-balancing/health-checks#http-health-checks

**Terrafrom Mappings:**

  * `google_compute_http_health_check.name`

## Supported Methods

* `GET`: Get a gcp-compute-http-health-check by its "name"
* `LIST`: List all gcp-compute-http-health-check
* ~~`SEARCH`~~