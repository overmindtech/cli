---
title: CloudFront Continuous Deployment Policy
sidebar_label: cloudfront-continuous-deployment-policy
---

A CloudFront Continuous Deployment Policy is an Amazon CloudFront configuration object that allows you to shift viewer traffic between two CloudFront distributions (normally a _staging_ and a _production_ distribution) in a controlled, progressive way. By defining percentage-based traffic splits or header-based routing rules, you can carry out blue/green or canary releases, test new versions of your application, and roll back instantly if problems occur.  
Official documentation: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/continuous-deployment.html

## Supported Methods

- `GET`: Get a CloudFront Continuous Deployment Policy by ID
- `LIST`: List CloudFront Continuous Deployment Policies
- `SEARCH`: Search CloudFront Continuous Deployment Policies by ARN

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

DNS records (usually CNAME or ALIAS/ANAME) that point end-user domains to the target CloudFront distributions determine which viewers are subject to a continuous deployment policy. When a policy is enabled, those DNS entries still resolve to the same CloudFront hostnames, but the policy decides how the resulting requests are routed internally between the staging and production distributions. Overmind therefore links the policy to related DNS resources so you can trace which public hostnames—and consequently which users—are affected by a particular traffic-splitting setup.
