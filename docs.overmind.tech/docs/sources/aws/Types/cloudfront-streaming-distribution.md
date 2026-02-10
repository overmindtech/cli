---
title: CloudFront Streaming Distribution
sidebar_label: cloudfront-streaming-distribution
---

An Amazon CloudFront Streaming Distribution is a special type of CloudFront distribution optimised for on-demand media streaming (historically using the RTMP protocol) and for serving video content over HTTP/S from an origin such as Amazon S3 or an on-premises media server. It automatically places edge cache nodes close to viewers, reducing latency and bandwidth costs while providing scalability, encryption and access control options. For full details see the official AWS documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/distribution-streaming.html

**Terrafrom Mappings:**

- `aws_cloudfront_distribution.arn`
- `aws_cloudfront_distribution.id`

## Supported Methods

- `GET`: Get a Streaming Distribution by ID
- `LIST`: List Streaming Distributions
- `SEARCH`: Search Streaming Distributions by ARN

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

Each CloudFront Streaming Distribution is reachable via a unique domain name that ends in `cloudfront.net`, and may also be associated with custom CNAMEs. These domain names appear in DNS records that overmind can discover and connect to the distribution resource.
