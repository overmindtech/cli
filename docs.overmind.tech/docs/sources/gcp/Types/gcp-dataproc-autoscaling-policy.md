---
title: GCP Dataproc Autoscaling Policy
sidebar_label: gcp-dataproc-autoscaling-policy
---

A Google Cloud Dataproc Autoscaling Policy defines how a Dataproc cluster should automatically grow or shrink its worker and secondary-worker (pre-emptible) node groups in response to load. Policies specify minimum and maximum instance counts, cooldown periods, and scaling rules based on YARN memory or CPU utilisation, allowing clusters to meet workload demand while controlling cost. Once created at the project or region level, a policy can be referenced by any Dataproc cluster in that location. For more detail see the official documentation: https://cloud.google.com/dataproc/docs/concepts/configuring-clusters/autoscaling.

**Terrafrom Mappings:**

- `google_dataproc_autoscaling_policy.name`

## Supported Methods

- `GET`: Get a gcp-dataproc-autoscaling-policy by its "name"
- `LIST`: List all gcp-dataproc-autoscaling-policy
- ~~`SEARCH`~~
