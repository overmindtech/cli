---
title: EC2 Snapshot
sidebar_label: ec2-snapshot
---

An Amazon EBS (Elastic Block Store) snapshot is an incremental, point-in-time backup of an EBS volume. Snapshots are stored in Amazon S3 and can be used to restore the original volume, create new volumes in the same or different Availability Zones, and copy data across Regions. They form a key part of disaster-recovery and migration workflows, allowing users to preserve data durability and quickly re-provision storage.  
Official documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-creating-snapshot.html

## Supported Methods

- `GET`: Get a snapshot by ID
- `LIST`: List all snapshots
- `SEARCH`: Search snapshots by ARN

## Possible Links

### [`ec2-volume`](/sources/aws/Types/ec2-volume)

A snapshot is created from, and can later be used to recreate, an EBS volume. Overmind links each `ec2-snapshot` to the `ec2-volume` it originated from (and, where relevant, the volumes restored from it), enabling you to trace data lineage and understand the blast radius of any change to the underlying storage.
