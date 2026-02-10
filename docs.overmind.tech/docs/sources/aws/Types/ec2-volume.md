---
title: EC2 Volume
sidebar_label: ec2-volume
---

An Amazon Elastic Block Store (EBS) volume provides persistent block-level storage for use with Amazon EC2 instances. Volumes can be attached to a single instance at a time (or multiple instances when using Multi-Attach), and retain their data independently of the life-cycle of that instance. Sizes, performance characteristics and encryption settings are configurable, allowing teams to tailor storage to the workload’s needs. Full service behaviour is documented by AWS here: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSVolumes.html

**Terrafrom Mappings:**

- `aws_ebs_volume.id`

## Supported Methods

- `GET`: Get a volume by ID
- `LIST`: List all volumes
- `SEARCH`: Search volumes by ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

A volume may be attached to, detached from or created alongside an EC2 instance. Overmind links the two resources so you can trace how storage changes could affect, or be affected by, the compute resource that consumes it.
