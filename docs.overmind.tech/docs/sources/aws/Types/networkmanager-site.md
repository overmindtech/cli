---
title: Networkmanager Site
sidebar_label: networkmanager-site
---

An AWS Network Manager **Site** represents a real-world location—such as a corporate office, data centre or colocation facility—that forms part of an organisation’s Global Network. It provides the context in which devices and network links are deployed, enabling AWS Network Manager to map physical geography to logical connectivity. For a full description of the resource and its attributes, see the official AWS documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_Site.html

**Terrafrom Mappings:**

- `aws_networkmanager_site.arn`

## Supported Methods

- `GET`: Get a Networkmanager Site
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager Sites by GlobalNetworkId or Site ARN

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

A Site is always created within a single Global Network. The `GlobalNetworkId` on the Site identifies its parent `networkmanager-global-network`, forming a one-to-many relationship (one Global Network, many Sites).

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

Links represent individual network connections (e.g., MPLS, broadband) that terminate at a Site. Each `networkmanager-link` includes the `SiteId` of the Site where the connection is installed, so multiple Links can be related to one Site.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

Devices such as routers, firewalls or SD-WAN appliances are housed at a Site. Every `networkmanager-device` records the `SiteId` where it resides, creating a one-to-many relationship between a Site and its Devices.
