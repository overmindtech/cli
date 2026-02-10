---
title: Reserved EC2 Instance
sidebar_label: ec2-reserved-instance
---

An AWS Reserved EC2 Instance represents a pre-paid or partially pre-paid commitment to run a specific instance type in a given Availability Zone or Region for a fixed term (one or three years). By committing up-front, you can obtain a significant discount compared with on-demand pricing, but you also take on the risk of paying for capacity you might not end up using. Overmind treats each Reserved Instance as its own resource so that you can surface any financial or capacity-planning risk associated with your reservation portfolio before a deployment is made.  
For detailed information on how Reserved Instances work, see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/reserved-instances.html

## Supported Methods

- `GET`: Get a reserved EC2 instance by ID
- `LIST`: List all reserved EC2 instances
- `SEARCH`: Search reserved EC2 instances by ARN
