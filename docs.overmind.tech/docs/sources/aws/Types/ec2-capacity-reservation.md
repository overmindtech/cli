---
title: Capacity Reservation
sidebar_label: ec2-capacity-reservation
---

An Amazon EC2 Capacity Reservation is an AWS construct that sets aside compute capacity for one or more instance types in a specific Availability Zone, guaranteeing that the reserved capacity is available whenever you need to launch instances. Capacity Reservations can be created individually or as members of a Capacity Reservation Fleet, allowing you to reserve capacity across several instance types and Zones in a single request. This is particularly useful for workloads that must start at short notice, seasonal traffic peaks, or disaster-recovery scenarios.  
For a detailed explanation, refer to the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-capacity-reservations.html

**Terrafrom Mappings:**

- `aws_ec2_capacity_reservation_fleet.id`

## Supported Methods

- `GET`: Get a capacity reservation fleet by ID
- `LIST`: List capacity reservation fleets
- `SEARCH`: Search capacity reservation fleets by ARN

## Possible Links

### [`ec2-placement-group`](/sources/aws/Types/ec2-placement-group)

A Capacity Reservation can be scoped to a placement group. When the `placement_group_arn` (or equivalent Terraform argument) is specified, Overmind links the reservation to that placement group so you can see how the reserved capacity aligns with your low-latency or HPC topology.

### [`ec2-capacity-reservation-fleet`](/sources/aws/Types/ec2-capacity-reservation-fleet)

If the reservation was created as part of a Capacity Reservation Fleet, Overmind links it to its parent fleet. This lets you trace individual reservations back to the fleet that manages them and understand how they contribute to the overall pool of reserved capacity across instance types and Availability Zones.
