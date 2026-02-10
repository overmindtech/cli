---
title: GCP Compute Health Check
sidebar_label: gcp-compute-health-check
---

A **GCP Compute Health Check** is a monitored probe that periodically tests the reachability and responsiveness of Google Cloud resources—such as VM instances, managed instance groups, or back-ends behind a load balancer—and reports their health status. These checks allow Google Cloud’s load balancers and auto-healing mechanisms to route traffic only to healthy instances, improving service reliability and availability. You can configure different protocols (HTTP, HTTPS, TCP, SSL, or HTTP/2), thresholds, and time-outs to suit your workload’s requirements.  
For full details, see the official documentation: https://cloud.google.com/load-balancing/docs/health-checks

**Terrafrom Mappings:**

- `google_compute_health_check.name`

## Supported Methods

- `GET`: Get GCP Compute Health Check by "gcp-compute-health-check-name"
- `LIST`: List all GCP Compute Health Check items
- ~~`SEARCH`~~
