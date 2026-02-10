---
title: CloudFront Origin Request Policy
sidebar_label: cloudfront-origin-request-policy
---

A CloudFront Origin Request Policy defines which HTTP headers, cookies and query-string parameters Amazon CloudFront passes from the edge to your origin. By attaching a policy to a cache behaviour you can standardise the information that reaches your origin, independent of any caching decisions. Policies are reusable across multiple distributions, making configuration simpler and less error-prone.
For further details refer to the [AWS documentation](https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_OriginRequestPolicy.html).

**Terrafrom Mappings:**

- `aws_cloudfront_origin_request_policy.id`

## Supported Methods

- `GET`: Get Origin Request Policy by ID
- `LIST`: List Origin Request Policies
- `SEARCH`: Origin Request Policy by ARN
