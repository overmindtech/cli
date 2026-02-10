---
title: EC2 Instance Event Window
sidebar_label: ec2-instance-event-window
---

An EC2 Instance Event Window is an Amazon EC2 scheduling feature that lets you specify one or more preferred time ranges during which planned AWS maintenance events (for example, a reboot, stop/start or software update) may be applied to your instances. By defining event windows, you retain greater control over when service-initiated interruptions occur, enabling you to align maintenance with your own change-management processes and minimise unplanned impact.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/event-windows.html

## Supported Methods

- `GET`: Get an event window by ID
- `LIST`: List all event windows
- `SEARCH`: Search for event windows by ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

An event window can be associated with one or more EC2 instances. When a linkage exists, those instances will only receive scheduled maintenance events during the time ranges defined in the referenced EC2 Instance Event Window.
