---
title: GCP Dns Managed Zone
sidebar_label: gcp-dns-managed-zone
---

A Cloud DNS Managed Zone is a logical container within Google Cloud that holds the DNS records for a particular namespace (for example, `example.com`). Each managed zone is served by a set of authoritative name servers and can be either public (resolvable on the public internet) or private (resolvable only from selected VPC networks). Managed zones let you create, update, and delete DNS resource-record sets using the Cloud DNS API, gcloud CLI, or Terraform.
For full details see Google’s documentation: https://docs.cloud.google.com/dns/docs/zones

**Terrafrom Mappings:**

- `google_dns_managed_zone.name`

## Supported Methods

- `GET`: Get a gcp-dns-managed-zone by its "name"
- `LIST`: List all gcp-dns-managed-zone
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Private managed zones can be attached to one or more VPC networks. When such a link exists, DNS queries originating from resources inside the referenced `gcp-compute-network` are resolved using the records defined in the managed zone. Overmind surfaces this relationship to show which networks will be affected by changes to the zone’s records or visibility settings.

### [`gcp-container-cluster`](/sources/gcp/Types/gcp-container-cluster)

Google Kubernetes Engine may automatically create or rely on Cloud DNS managed zones for features such as service discovery, Cloud DNS-based Pod/Service FQDN resolution, or workload identity federation. Linking a `gcp-dns-managed-zone` to a `gcp-container-cluster` allows Overmind to highlight how DNS adjustments could impact cluster-internal name resolution or ingress behaviour for that cluster.
