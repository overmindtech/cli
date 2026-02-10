---
title: EC2 Instance Status
sidebar_label: ec2-instance-status
---

An EC2 Instance Status record summarises the current health of a running Amazon Elastic Compute Cloud (EC2) instance. AWS performs two types of status checks—system checks (that assess the underlying host and network) and instance checks (that confirm the guest operating system is reachable). Together they indicate whether the instance is able to accept traffic and function as expected.
Overmind ingests these status objects so that you can surface potential availability risks (e.g. persistent instance check failures) before promoting or modifying a deployment.
For a detailed explanation of how AWS generates and interprets these checks, see the [official AWS documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/status-checks.html).

## Supported Methods

- `GET`: Get an EC2 instance status by Instance ID
- `LIST`: List all EC2 instance statuses
- `SEARCH`: Search EC2 instance statuses by ARN
