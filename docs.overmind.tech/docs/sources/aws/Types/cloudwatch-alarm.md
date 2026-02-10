---
title: CloudWatch Alarm
sidebar_label: cloudwatch-alarm
---

An Amazon CloudWatch Alarm watches a single CloudWatch metric (or a maths expression based on one or more metrics) and performs one or more actions when the metric breaches a threshold for a specified number of evaluation periods. Typical actions include sending an SNS notification, invoking an Auto Scaling policy or stopping, terminating, rebooting or recovering an EC2 instance. Alarms are therefore often a critical part of operational resilience and cost-control strategies, and mis-configuration can lead to missed incidents or unwanted automated actions.  
Official documentation: https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/AlarmThatSendsEmail.html

**Terraform Mappings:**

- `aws_cloudwatch_metric_alarm.alarm_name`

## Supported Methods

- `GET`: Get an alarm by name
- `LIST`: List all alarms
- `SEARCH`: Search for alarms. This accepts JSON in the format of `cloudwatch.DescribeAlarmsForMetricInput`
