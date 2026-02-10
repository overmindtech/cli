---
title: CloudFront Realtime Log Config
sidebar_label: cloudfront-realtime-log-config
---

Amazon CloudFront Realtime Log Configs define the structure of the near-real-time log data that CloudFront can stream to a destination such as Kinesis Data Streams. A Realtime Log Config specifies which data fields are captured, the sampling rate, and the endpoint to which the records are delivered. This enables teams to observe viewer requests, latency, cache behaviour and other metrics with sub-second visibility, allowing faster troubleshooting and performance tuning.  
For a detailed description of the service and its capabilities, refer to the official AWS documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/real-time-logs.html

**Terrafrom Mappings:**

- `aws_cloudfront_realtime_log_config.arn`

## Supported Methods

- `GET`: Get Realtime Log Config by Name
- `LIST`: List Realtime Log Configs
- `SEARCH`: Search Realtime Log Configs by ARN
