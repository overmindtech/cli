---
title: GCP Compute Reservation
sidebar_label: gcp-compute-reservation
---

A GCP Compute Reservation is a zonal reservation of Compute Engine capacity that guarantees the availability of a specific machine type (and, optionally, attached GPUs, local SSDs, etc.) for when you later launch virtual machine (VM) instances. By pre-allocating vCPU and memory resources, reservations help you avoid capacity-related scheduling failures in busy zones and can be shared across projects inside the same organisation if desired. See the official documentation for full details: https://docs.cloud.google.com/compute/docs/instances/reservations-overview.

**Terrafrom Mappings:**

- `google_compute_reservation.name`

## Supported Methods

- `GET`: Get GCP Compute Reservation by "gcp-compute-reservation-name"
- `LIST`: List all GCP Compute Reservation items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-region-commitment`](/sources/gcp/Types/gcp-compute-region-commitment)

Capacity held by a reservation counts against any existing regional commitment in the same region. By linking a reservation to its corresponding `gcp-compute-region-commitment`, you can see whether the reserved resources are already discounted or whether additional commitments may be required.
