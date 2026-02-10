---
title: GCP Compute Autoscaler
sidebar_label: gcp-compute-autoscaler
---

The Google Cloud Compute Autoscaler is a regional or zonal resource that automatically adds or removes VM instances from a Managed Instance Group in response to workload demand. By scaling on CPU utilisation, load-balancing capacity, Cloud Monitoring metrics, or pre-defined schedules, it helps keep applications responsive while keeping infrastructure spending under control.  
For detailed information, consult the official documentation: https://cloud.google.com/compute/docs/autoscaler

**Terrafrom Mappings:**

- `google_compute_autoscaler.name`

## Supported Methods

- `GET`: Get GCP Compute Autoscaler by "gcp-compute-autoscaler-name"
- `LIST`: List all GCP Compute Autoscaler items
- ~~`SEARCH`~~
