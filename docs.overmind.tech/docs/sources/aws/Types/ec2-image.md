---
title: Amazon Machine Image (AMI)
sidebar_label: ec2-image
---

An Amazon Machine Image (AMI) is a pre-configured, read-only template that defines the software stack required to launch an Amazon EC2 instance. It typically contains an operating system, application server, and any additional software or configuration needed for your workload. By selecting or creating an AMI you can reproduce identical instances at scale, roll back to known-good states, or share hardened golden images across accounts and Regions.  
For a full explanation of AMIs, see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html.

**Terrafrom Mappings:**

- `aws_ami.id`

## Supported Methods

- `GET`: Get an AMI by ID
- `LIST`: List all AMIs
- `SEARCH`: Search AMIs by ARN
