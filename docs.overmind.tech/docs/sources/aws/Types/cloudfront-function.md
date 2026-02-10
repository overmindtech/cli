---
title: CloudFront Function
sidebar_label: cloudfront-function
---

Amazon CloudFront Functions let you run lightweight JavaScript code at CloudFront edge locations, enabling real-time manipulation of HTTP requests and responses without the latency of invoking AWS Lambda. Typical use-cases include URL rewrites, header manipulation, access control and A/B testing, all executed in under one millisecond at every edge. For more detail see the official AWS documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cloudfront-functions.html

**Terrafrom Mappings:**

- `aws_cloudfront_function.name`

## Supported Methods

- `GET`: Get a CloudFront Function by name
- `LIST`: List CloudFront Functions
- `SEARCH`: Search CloudFront Functions by ARN
