---
title: GCP Dns Managed Zone
sidebar_label: gcp-dns-managed-zone
---

A Google Cloud DNS Managed Zone is a logical container for DNS resource records that share the same DNS name suffix. Managed zones can be configured as public (resolvable from the internet) or private (resolvable only from one or more selected VPC networks). They are the fundamental unit that Cloud DNS uses to host, serve and manage authoritative DNS data for your domains.  
Official documentation: https://cloud.google.com/dns/docs/zones

**Terrafrom Mappings:**

  * `google_dns_managed_zone.name`

## Supported Methods

* `GET`: Get a gcp-dns-managed-zone by its "name"
* `LIST`: List all gcp-dns-managed-zone
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Private managed zones are explicitly linked to one or more VPC networks. The association determines which networks can resolve the zone’s records, so an Overmind relationship helps surface reachability and leakage risks between a DNS zone and the networks that consume it.

### [`gcp-container-cluster`](/sources/gcp/Types/gcp-container-cluster)

GKE clusters frequently create or rely on Cloud DNS managed zones for service discovery and in-cluster load-balancing (e.g., when CloudDNS for Service Directory is enabled). Mapping a cluster to its managed zones reveals dependencies that affect name resolution, cross-cluster communication and potential namespace conflicts.