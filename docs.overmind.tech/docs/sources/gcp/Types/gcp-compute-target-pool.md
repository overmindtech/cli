---
title: GCP Compute Target Pool
sidebar_label: gcp-compute-target-pool
---

A Compute Target Pool is a regional resource that groups multiple VM instances so they can receive incoming traffic from legacy network TCP load balancers or be used as failover targets for forwarding rules. Target pools can also be linked to one or more Health Checks to determine the availability of their member instances. Official documentation: https://docs.cloud.google.com/load-balancing/docs/target-pools

**Terrafrom Mappings:**

- `google_compute_target_pool.id`

## Supported Methods

- `GET`: Get a gcp-compute-target-pool by its "name"
- `LIST`: List all gcp-compute-target-pool
- `SEARCH`: Search with full ID: projects/[project]/regions/[region]/targetPools/[name] (used for terraform mapping).

## Possible Links

### [`gcp-compute-health-check`](/sources/gcp/Types/gcp-compute-health-check)

A target pool may reference one or more Health Checks. These checks are executed against each instance in the pool to decide whether the instance should receive traffic. Overmind links a target pool to any health check resources it is configured to use.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Member virtual machines are registered in the target pool. Overmind establishes links from the target pool to every compute instance that is currently part of the pool.

### [`gcp-compute-target-pool`](/sources/gcp/Types/gcp-compute-target-pool)

Target pools can appear as dependencies of other target pools in scenarios such as cross-region failover configurations. Overmind represents these intra-type relationships with links between the relevant target pool resources.
