---
title: GCP Dataproc Autoscaling Policy
sidebar_label: gcp-dataproc-autoscaling-policy
---

A GCP Dataproc Autoscaling Policy defines the rules that Google Cloud Dataproc uses to automatically add or remove worker nodes from a Dataproc cluster in response to workload demand. By specifying target utilisation levels, cooldown periods, graceful decommissioning time-outs and per-node billing settings, the policy ensures that clusters expand to meet spikes in processing requirements and shrink when demand falls, optimising both performance and cost.  
For a full description of each field and the underlying API, see the official Google Cloud documentation: https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.autoscalingPolicies.

**Terraform Mappings:**

* `google_dataproc_autoscaling_policy.name`

## Supported Methods

* `GET`: Get a gcp-dataproc-autoscaling-policy by its "name"
* `LIST`: List all gcp-dataproc-autoscaling-policy
* ~~`SEARCH`~~
