---
title: Placement Group
sidebar_label: ec2-placement-group
---

An EC2 Placement Group is an AWS construct that lets you influence how Elastic Compute Cloud (EC2) instances are positioned on the underlying hardware. By creating a placement group with a strategy of `cluster`, `spread`, or `partition`, you can optimise for high-bandwidth, low-latency networking, reduce the risk of simultaneous hardware failures, or isolate groups of instances from one another. For full details, refer to the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/placement-groups.html

**Terrafrom Mappings:**

- `aws_placement_group.id`

## Supported Methods

- `GET`: Get a placement group by ID
- `LIST`: List all placement groups
- `SEARCH`: Search for placement groups by ARN
