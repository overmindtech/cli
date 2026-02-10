---
title: Capacity Reservation Fleet
sidebar_label: ec2-capacity-reservation-fleet
---

A Capacity Reservation Fleet is an Amazon EC2 resource that lets you create and manage a group of Capacity Reservations in a single operation. By specifying instance attributes such as instance types, platforms and Availability Zones, you can ensure that the compute capacity your workload requires will be held for you ahead of time, even during periods of high demand. This is especially useful when you need to guarantee that a heterogeneous mix of instances will be available at launch, for example during large-scale events or disaster-recovery drills.  
For more information, see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateCapacityReservationFleet.html

## Supported Methods

- `GET`: Get a capacity reservation fleet by ID
- `LIST`: List capacity reservation fleets
- `SEARCH`: Search capacity reservation fleets by ARN

## Possible Links

### [`ec2-capacity-reservation`](/sources/aws/Types/ec2-capacity-reservation)

A Capacity Reservation Fleet is essentially an umbrella object that owns one or more individual Capacity Reservations. Each linked `ec2-capacity-reservation` represents a single slice of capacity that was created as part of the fleet’s allocation strategy, and tracking these links lets you understand which reservations belong to which fleet and how capacity is distributed across instance types and Availability Zones.
