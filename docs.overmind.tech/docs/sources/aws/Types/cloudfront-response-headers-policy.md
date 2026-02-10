---
title: CloudFront Response Headers Policy
sidebar_label: cloudfront-response-headers-policy
---

A CloudFront Response Headers Policy is an AWS configuration object that specifies the HTTP response headers that Amazon CloudFront adds to, removes from, or overrides on the responses it returns to viewers. By defining a policy you can, for example, enforce security-related headers (such as `Strict-Transport-Security` or `Content-Security-Policy`), apply custom cache-control directives, or expose additional headers to browsers for client-side logic. Once created, a response headers policy can be associated with one or more CloudFront distributions, allowing consistent header behaviour across multiple delivery configurations.  
For full details see the AWS documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/response-headers-policies.html

**Terrafrom Mappings:**

- `aws_cloudfront_response_headers_policy.id`

## Supported Methods

- `GET`: Get Response Headers Policy by ID
- `LIST`: List Response Headers Policies
- `SEARCH`: Search Response Headers Policy by ARN
