---
title: Route53 Record Set
sidebar_label: route53-resource-record-set
---

A Route 53 Resource Record Set represents a single DNS record (or a group of records with the same name and type) that lives inside a specific hosted zone. It defines how Amazon Route 53 answers DNS queries for the associated domain name, including the record type (A, AAAA, CNAME, MX, TXT, SRV, etc.), routing policy (simple, weighted, latency, geolocation, fail-over, multi-value, or alias), time-to-live (TTL) and, optionally, a linked health check.
For full details see the AWS documentation: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/RRSet.html

**Terrafrom Mappings:**

- `aws_route53_record.arn`
- `aws_route53_record.id`

## Supported Methods

- `GET`: Get a resource record set. The ID is the concatenation of the hosted zone, name, and record type (`{hostedZone}.{name}.{type}`)
- `LIST`: List all resource record sets

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

Because a Route 53 record set ultimately becomes a DNS record that can be queried on the public or private internet, each record set naturally maps to an Overmind `dns` item. Following this link lets you see the vendor-agnostic representation of the record (name, type, TTL and value) and how it is consumed by other infrastructure components.

### [`route53-health-check`](/sources/aws/Types/route53-health-check)

If the record set is configured with a fail-over, latency, or weighted routing policy that refers to a Route 53 health check, Overmind links the record set to that `route53-health-check` item. This shows the dependency between DNS resolution and the health status of the monitored endpoint, helping you understand how an unhealthy resource could affect name resolution.
