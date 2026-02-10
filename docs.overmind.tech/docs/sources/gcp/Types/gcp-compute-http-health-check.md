---
title: GCP Compute Http Health Check
sidebar_label: gcp-compute-http-health-check
---

A **Google Cloud Compute HTTP Health Check** is a legacy, regional health-check resource that periodically issues HTTP `GET` requests to a specified path on your instances or load-balanced back-ends. If an instance responds with an acceptable status code (e.g. `200–299`) within the configured timeout for the required number of consecutive probes, it is marked healthy; otherwise, it is marked unhealthy. Load balancers and target pools use this signal to route traffic only to healthy instances, helping to maintain application availability.  
Google now recommends the newer, unified _Health Check_ resource for most use-cases, but HTTP Health Checks remain fully supported and are still encountered in many estates.  
For full details, see the official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/httpHealthChecks

**Terrafrom Mappings:**

- `google_compute_http_health_check.name`

## Supported Methods

- `GET`: Get a gcp-compute-http-health-check by its "name"
- `LIST`: List all gcp-compute-http-health-check
- ~~`SEARCH`~~
