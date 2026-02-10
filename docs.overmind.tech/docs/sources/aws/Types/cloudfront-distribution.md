---
title: CloudFront Distribution
sidebar_label: cloudfront-distribution
---

Amazon CloudFront Distributions are globally-replicated configurations that tell the CloudFront CDN how to cache and deliver your content to end-users. Each distribution defines one or more origins, cache behaviours, security settings and optional edge-compute integrations. See the official AWS documentation for a full description: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/distribution-working-with.html

**Terrafrom Mappings:**

- `aws_cloudfront_distribution.arn`

## Supported Methods

- `GET`: Get a distribution by ID
- `LIST`: List all distributions
- `SEARCH`: Search distributions by ARN

## Possible Links

### [`cloudfront-key-group`](/sources/aws/Types/cloudfront-key-group)

A distribution can reference one or more Key Groups in its `TrustedKeyGroups` configuration to validate signed URLs or signed cookies. If a Key Group ID appears in the distribution’s config, Overmind links the two.

### [`cloudfront-continuous-deployment-policy`](/sources/aws/Types/cloudfront-continuous-deployment-policy)

Distributions may have an attached Continuous Deployment Policy (`ContinuousDeploymentPolicyId`) that allows blue/green traffic shifting. Overmind links the distribution to that policy.

### [`cloudfront-cache-policy`](/sources/aws/Types/cloudfront-cache-policy)

Every cache behaviour in a distribution can specify a `CachePolicyId`. Overmind links the distribution to any Cache Policies it relies on.

### [`cloudfront-function`](/sources/aws/Types/cloudfront-function)

Viewer request / response CloudFront Functions can be associated with behaviours in the distribution. Those references create links between the distribution and the function resources.

### [`cloudfront-origin-request-policy`](/sources/aws/Types/cloudfront-origin-request-policy)

Behaviours can also specify an `OriginRequestPolicyId` that controls which headers, cookies and query strings are sent to the origin. Overmind links distributions to the referenced Origin Request Policies.

### [`cloudfront-realtime-log-config`](/sources/aws/Types/cloudfront-realtime-log-config)

If real-time logging is enabled, the distribution contains one or more `RealtimeLogConfigArn` values. Overmind uses those to link the distribution to its real-time log configuration.

### [`cloudfront-response-headers-policy`](/sources/aws/Types/cloudfront-response-headers-policy)

Behaviours may include a `ResponseHeadersPolicyId` that injects security or custom headers. Overmind links the distribution to the associated Response Headers Policies.

### [`dns`](/sources/stdlib/Types/dns)

Public access to a distribution is normally via the CloudFront domain name or an alias/CNAME such as `www.example.com`. When a DNS record (e.g., Route 53 ALIAS) targets the distribution’s domain, Overmind links the DNS record to the distribution.

### [`lambda-function`](/sources/aws/Types/lambda-function)

Lambda@Edge functions (standard Lambda functions replicated to edge locations) can be attached to behaviours for request or response processing. These associations create links between the distribution and the Lambda functions.

### [`s3-bucket`](/sources/aws/Types/s3-bucket)

An S3 bucket is commonly used as an origin. When the distribution’s origin points at an S3 bucket domain or ARN, Overmind links the distribution to that bucket.
