---
title: Job
sidebar_label: Job
---

A Kubernetes Job is a controller that runs one-off or batch tasks to completion. It creates one or more Pods and tracks their execution until the specified number have finished successfully. Jobs are ideal for database migrations, data processing, or any workload that needs to run to completion rather than persist indefinitely. A Job retries failed Pods according to its back-off policy and is marked as complete once all Pods exit successfully. For more details, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/job/

**Terrafrom Mappings:**

- `kubernetes_job.metadata[0].name`
- `kubernetes_job_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Job by name
- `LIST`: List all Jobs
- `SEARCH`: Search for a Job using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Pod`](/sources/k8s/Types/Pod)

A Job spawns one or more Pods to run its workload; each Pod created by the Job is linked back to it via the Job’s `ownerReferences` metadata.
