---
title: GCP Compute Public Delegated Prefix
sidebar_label: gcp-compute-public-delegated-prefix
---

A Google Cloud Compute Public Delegated Prefix represents a block of publicly-routable IPv4 or IPv6 addresses that Google has reserved and delegated to your project in a given region. Once the prefix exists you can further subdivide it into smaller delegated prefixes or assign individual addresses to resources such as VM instances, forwarding rules, or load balancers. Public Delegated Prefixes enable you to bring your own IP space, ensure predictable address allocation and control how traffic enters your network.
Official documentation: https://docs.cloud.google.com/vpc/docs/create-pdp

**Terrafrom Mappings:**

- `google_compute_public_delegated_prefix.id`

## Supported Methods

- `GET`: Get a gcp-compute-public-delegated-prefix by its "name"
- `LIST`: List all gcp-compute-public-delegated-prefix
- `SEARCH`: Search with full ID: projects/[project]/regions/[region]/publicDelegatedPrefixes/[name] (used for terraform mapping).

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

A Public Delegated Prefix is created within, and therefore belongs to, a specific Cloud Resource Manager project. The project provides billing, IAM, and quota context for the prefix.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

A Public Delegated Prefix can itself be the parent of smaller delegated prefixes; these child prefixes are represented by additional `gcp-compute-public-delegated-prefix` resources that reference the parent block.
