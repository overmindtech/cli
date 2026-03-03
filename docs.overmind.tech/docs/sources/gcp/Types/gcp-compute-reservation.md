---
title: GCP Compute Reservation
sidebar_label: gcp-compute-reservation
---

A GCP Compute Reservation is a zonal capacity-planning resource that lets you pre-allocate Compute Engine virtual machine capacity so that it is always available when your workloads need it. By creating a reservation you can guarantee that the required number and type of vCPUs, memory and accelerators are held for your project in a particular zone, avoiding scheduling failures during peaks or regional outages. For full details, see the official Google Cloud documentation: https://cloud.google.com/compute/docs/instances/reserving-zonal-resources

**Terrafrom Mappings:**

- `google_compute_reservation.name`

## Supported Methods

- `GET`: Get GCP Compute Reservation by "gcp-compute-reservation-name"
- `LIST`: List all GCP Compute Reservation items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-region-commitment`](/sources/gcp/Types/gcp-compute-region-commitment)

Reservations guarantee capacity, while regional commitments provide sustained-use discounts for that capacity. A reservation created in a zone may be covered by, or contribute to the utilisation of, a regional commitment in the same region, so analysing the commitment alongside the reservation reveals both availability and cost-optimisation aspects of the deployment.
