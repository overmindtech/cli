---
title: GCP Compute Region Commitment
sidebar_label: gcp-compute-region-commitment
---

A GCP Compute Region Commitment is an agreement in which you purchase a predefined amount of vCPU, memory or GPU capacity in a specific region for a fixed term (one or three years) in return for a reduced hourly price. Commitments are applied automatically to matching usage within the chosen region, helping to lower running costs while guaranteeing a baseline level of capacity. For a detailed explanation of the feature, see the official documentation: https://docs.cloud.google.com/compute/docs/reference/rest/v1/regionCommitments/list.

**Terrafrom Mappings:**

- `google_compute_region_commitment.name`

## Supported Methods

- `GET`: Get a gcp-compute-region-commitment by its "name"
- `LIST`: List all gcp-compute-region-commitment
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-reservation`](/sources/gcp/Types/gcp-compute-reservation)

A region commitment can be consumed by one or more compute reservations in the same region. When a reservation launches virtual machine instances, the resources they use are first drawn from any applicable commitments so that the discounted commitment pricing is applied automatically.
