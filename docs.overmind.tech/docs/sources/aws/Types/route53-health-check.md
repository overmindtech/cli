---
title: Route53 Health Check
sidebar_label: route53-health-check
---

Amazon Route 53 health checks continuously monitor the availability and latency of your application endpoints (such as web servers, API gateways or other resources) and can automatically trigger DNS fail-over when an endpoint becomes unhealthy. Each health check can also be configured to integrate with Amazon CloudWatch, enabling alerting and automation based on the current health state.  
For full details, refer to the official AWS documentation: [Amazon Route 53 Health Checks](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-failover.html).

**Terrafrom Mappings:**

- `aws_route53_health_check.id`

## Supported Methods

- `GET`: Get health check by ID
- `LIST`: List all health checks
- `SEARCH`: Search for health checks by ARN

## Possible Links

### [`cloudwatch-alarm`](/sources/aws/Types/cloudwatch-alarm)

A CloudWatch alarm can be created that uses the `HealthCheckStatus` metric emitted for a specific Route 53 health check. This allows the alarm to publish notifications or trigger automated responses whenever the health check reports an unhealthy or healthy state. Overmind therefore records a link from a Route 53 health check to any CloudWatch alarms that reference its ID so you can immediately see which alarms will fire if the check changes status.
