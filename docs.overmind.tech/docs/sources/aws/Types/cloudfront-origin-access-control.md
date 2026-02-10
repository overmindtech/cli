---
title: Cloudfront Origin Access Control
sidebar_label: cloudfront-origin-access-control
---

Amazon CloudFront Origin Access Control (OAC) is a security feature that allows you to restrict access to the origin of a CloudFront distribution, ensuring that all requests are authenticated and authorised by CloudFront before reaching your S3 bucket, Application Load Balancer, or custom origin. OAC is the modern replacement for Origin Access Identities (OAI) and supports both SigV4‐signed requests and IAM authentication, giving you more granular control over how CloudFront communicates with your back-end resources. By configuring an OAC you prevent direct exposure of your origin on the public internet, helping to mitigate data-exfiltration and origin-based attacks.  
For further information see the official AWS documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-origin.html#concept-origin-access-control

**Terrafrom Mappings:**

- `aws_cloudfront_origin_access_control.id`

## Supported Methods

- `GET`: Get Origin Access Control by ID
- `LIST`: List Origin Access Controls
- `SEARCH`: Origin Access Control by ARN
