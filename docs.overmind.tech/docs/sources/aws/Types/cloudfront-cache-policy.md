---
title: CloudFront Cache Policy
sidebar_label: cloudfront-cache-policy
---

An AWS CloudFront Cache Policy specifies the rules that dictate how CloudFront caches HTTP responses at edge locations. It determines which headers, cookies and query-string parameters are included in the cache key, how long objects remain in the cache (TTL values), and whether to compress the response before it is served to viewers. By creating and attaching custom cache policies to distributions or behaviours, you can fine-tune cache efficiency, control origin load, and optimise performance for different types of content. For a full description of the resource and its attributes, refer to the [AWS documentation](https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_CachePolicy.html).

**Terrafrom Mappings:**

- `aws_cloudfront_cache_policy.id`

## Supported Methods

- `GET`: Get a CloudFront Cache Policy
- `LIST`: List CloudFront Cache Policies
- `SEARCH`: Search CloudFront Cache Policies by ARN
