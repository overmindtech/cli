---
title: GCP Compute Target Pool
sidebar_label: gcp-compute-target-pool
---

A Google Cloud Compute Target Pool is a regional grouping of VM instances that acts as the backend for the legacy TCP/UDP network load balancer. The pool defines which instances receive traffic, the optional session-affinity policy, the associated health checks that determine instance health, and an optional fail-over target pool for backup. See the official documentation for full details: https://cloud.google.com/compute/docs/reference/rest/v1/targetPools

**Terrafrom Mappings:**

  * `google_compute_target_pool.id`

## Supported Methods

* `GET`: Get a gcp-compute-target-pool by its "name"
* `LIST`: List all gcp-compute-target-pool
* `SEARCH`: Search with full ID: projects/[project]/regions/[region]/targetPools/[name] (used for terraform mapping).

## Possible Links

### [`gcp-compute-health-check`](/sources/gcp/Types/gcp-compute-health-check)

A target pool may reference one or more health checks through its `healthChecks` field. These health checks are used by Google Cloud to probe the instances in the pool and decide whether traffic should be sent to a particular VM.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Each target pool contains a list of VM instances (`instances` field) that will receive load-balanced traffic. Overmind links the pool to every instance it contains.

### [`gcp-compute-target-pool`](/sources/gcp/Types/gcp-compute-target-pool)

A target pool can specify another target pool as its `backupPool` to provide fail-over capacity, and it can itself be referenced as a backup by other pools. Overmind surfaces these peer-to-peer relationships between target pools.