---
title: Hosted Zone
sidebar_label: route53-hosted-zone
---

An Amazon Route 53 hosted zone is a container for all of the DNS records that belong to a single domain (for example `example.com`) or a sub-domain. It represents a DNS namespace within Route 53 and is the primary object you create when you want AWS to answer queries for your domain. Hosted zones can be public (resolving queries on the public Internet) or private (resolving only within one or more associated VPCs), and support advanced features such as DNSSEC signing and alias records to AWS resources.  
For full details see the AWS documentation: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html

**Terrafrom Mappings:**

- `aws_route53_hosted_zone_dnssec.id`
- `aws_route53_zone.zone_id`
- `aws_route53_zone_association.zone_id`

## Supported Methods

- `GET`: Get a hosted zone by ID
- `LIST`: List all hosted zones
- `SEARCH`: Search for a hosted zone by ARN

## Possible Links

### [`route53-resource-record-set`](/sources/aws/Types/route53-resource-record-set)

Each hosted zone contains one or more resource record sets. Overmind establishes a link from a Hosted Zone item to the `route53-resource-record-set` items that reside within it, allowing you to explore every DNS record that will be created, modified or deleted as part of a deployment.
