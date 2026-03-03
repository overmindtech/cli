---
title: GCP Compute Region Commitment
sidebar_label: gcp-compute-region-commitment
---

A Compute Region Commitment in Google Cloud Platform (GCP) represents a contractual agreement to purchase a certain amount of vCPU, memory, GPUs or local SSD capacity within a specific region for one or three years. In exchange for this up-front commitment, you receive a discounted hourly rate for the covered resources, regardless of whether the capacity is actually in use. Commitments are created per-project and per-region, and the discount automatically applies to any eligible VM instances running in that region. For full details see the official documentation: https://cloud.google.com/compute/docs/instances/signing-up-committed-use-discounts

**Terrafrom Mappings:**

- `google_compute_region_commitment.name`

## Supported Methods

- `GET`: Get a gcp-compute-region-commitment by its "name"
- `LIST`: List all gcp-compute-region-commitment
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-reservation`](/sources/gcp/Types/gcp-compute-reservation)

Reservations and commitments often work together: a reservation guarantees that capacity is available, while a commitment provides a discount for that capacity. When Overmind discovers a region commitment it links it to any compute reservations in the same project and region so you can see both the cost commitment and the capacity guarantee in one place.
