---
title: GCP Compute Autoscaler
sidebar_label: gcp-compute-autoscaler
---

A GCP Compute Autoscaler is a zonal or regional resource that automatically adds or removes VM instances from a managed instance group to keep your application running at the desired performance level and cost. Scaling decisions can be driven by policies based on average CPU utilisation, HTTP load-balancing capacity, Cloud Monitoring metrics, schedules, or per-instance utilisation. Full details can be found in the official documentation: https://cloud.google.com/compute/docs/autoscaler

**Terrafrom Mappings:**

- `google_compute_autoscaler.name`

## Supported Methods

- `GET`: Get GCP Compute Autoscaler by "gcp-compute-autoscaler-name"
- `LIST`: List all GCP Compute Autoscaler items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-instance-group-manager`](/sources/gcp/Types/gcp-compute-instance-group-manager)

Every autoscaler is attached to exactly one managed instance group; in the GCP API this relationship is expressed through the `target` field, which points to the relevant `instanceGroupManager` resource. Following this link in Overmind reveals which VM instances the autoscaler is responsible for scaling.
