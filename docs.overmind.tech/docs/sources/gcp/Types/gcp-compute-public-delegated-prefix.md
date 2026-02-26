---
title: GCP Compute Public Delegated Prefix
sidebar_label: gcp-compute-public-delegated-prefix
---

A Public Delegated Prefix is a regional IPv4 or IPv6 address range that you reserve from Google Cloud and can then subdivide and delegate to other projects, VPC networks, or Private Service Connect service attachments. It allows you to keep ownership of the parent prefix while giving consumers controlled use of sub-prefixes, simplifying address management and avoiding manual peering or routing configurations.  
For full details, see the official documentation: https://cloud.google.com/vpc/docs/create-pdp

**Terrafrom Mappings:**

  * `google_compute_public_delegated_prefix.id`

## Supported Methods

* `GET`: Get a gcp-compute-public-delegated-prefix by its "name"
* `LIST`: List all gcp-compute-public-delegated-prefix
* `SEARCH`: Search with full ID: projects/[project]/regions/[region]/publicDelegatedPrefixes/[name] (used for terraform mapping).

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

This prefix belongs to and is created within a specific Google Cloud project; the link points from the Public Delegated Prefix to its parent project.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

A parent Public Delegated Prefix can be linked to child delegated sub-prefixes (or vice-versa) to represent hierarchy and inheritance of the IP space.