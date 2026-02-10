---
title: Networkmanager Link
sidebar_label: networkmanager-link
---

An AWS Network Manager **Link** represents a physical or logical connection (for example, an MPLS circuit, Direct Connect connection, broadband, or internet link) that provides connectivity at a specific site within a global network. Links are used by Network Manager to calculate network health, aggregate telemetry and visualise topology. Each link is created inside a Site, and therefore inside a Global Network, and can later be associated with one or more network devices.  
Official documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_Link.html

**Terrafrom Mappings:**

- `aws_networkmanager_link.arn`

## Supported Methods

- `GET`: Get a Networkmanager Link
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager Links by GlobalNetworkId, GlobalNetworkId with SiteId, or ARN

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

A Link is a component of a single Global Network; this edge points from the Link to the Global Network that owns it.

### [`networkmanager-link-association`](/sources/aws/Types/networkmanager-link-association)

A Link can be associated with one or more devices. These associations are represented by Network Manager Link Association resources, which reference the Link as their parent.

### [`networkmanager-site`](/sources/aws/Types/networkmanager-site)

Every Link resides in exactly one Site; this relationship shows which Site the Link belongs to.

### [`networkmanager-network-resource-relationship`](/sources/aws/Types/networkmanager-network-resource-relationship)

Network Manager records discovered relationships between Links and other network resources (for example, AWS Transit Gateway attachments). This edge captures those discovered `network-resource-relationship` objects that involve the Link.
