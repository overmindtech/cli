---
title: Cron Job
sidebar_label: CronJob
---

A Kubernetes **CronJob** is a higher-level controller responsible for running a Job object on a repeating schedule, expressed in standard _cron_ syntax. It is typically used for routine, time-based tasks such as database backups, report generation, and regular housekeeping activities inside a cluster. The controller automatically creates the underlying Job at the scheduled time, monitors its execution and, depending on the configuration, retains or cleans up finished Jobs and their Pods. For a full description of the resource’s behaviour and available fields, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/

**Terrafrom Mappings:**

- `kubernetes_cron_job_v1.metadata[0].name`
- `kubernetes_cron_job.metadata[0].name`

## Supported Methods

- `GET`: Get a Cron Job by name
- `LIST`: List all Cron Jobs
- `SEARCH`: Search for a Cron Job using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
