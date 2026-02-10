---
title: Launch Template
sidebar_label: ec2-launch-template
---

An EC2 Launch Template is an AWS resource that stores the complete configuration needed to spin up one or more Amazon EC2 instances, including AMI ID, instance type, network settings, user-data scripts, and optional purchasing options such as Spot or On-Demand. By saving these parameters in a versioned template, teams can reproduce environments consistently, roll back to previous configurations, and simplify autoscaling and fleet operations.  
Official documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-launch-templates.html

**Terrafrom Mappings:**

- `aws_launch_template.id`

## Supported Methods

- `GET`: Get a launch template by ID
- `LIST`: List all launch templates
- `SEARCH`: Search for launch templates by ARN
