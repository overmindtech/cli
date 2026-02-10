---
title: EC2 Volume Status
sidebar_label: ec2-volume-status
---

The EC2 Volume Status resource represents the health information that AWS exposes for every Amazon Elastic Block Store (EBS) volume. Derived from the `DescribeVolumeStatus` API call, it records the results of automated status checks, any events that might affect I/O, and recommended user actions. Monitoring these objects in Overmind lets you spot degraded or impaired volumes before they compromise a deployment.  
For a complete description of the data returned by AWS, see the official documentation: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVolumeStatus.html

## Supported Methods

- `GET`: Get a volume status by volume ID
- `LIST`: List all volume statuses
- `SEARCH`: Search for volume statuses by ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

A Volume Status relates to the EC2 instance that the underlying EBS volume is currently attached to, if any. Overmind links the status object to the instance so you can trace how a failing or impaired volume might impact the workloads running on that instance.
