---
title: Networkmanager LinkAssociation
sidebar_label: networkmanager-link-association
---

A Network Manager **Link Association** represents the attachment of a specific physical or logical network **link** (for example, a DIA, MPLS or broadband circuit) to a **device** (such as a router, firewall, SD-WAN appliance) that resides at a site in an AWS Cloud WAN / Network Manager **global network**.  
Each association records which device terminates the link, the site it belongs to, bandwidth details and the operational state of that attachment.  
Official AWS documentation:  
https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_LinkAssociation.html

## Supported Methods

- `GET`: Get a Networkmanager Link Association
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager Link Associations by GlobalNetworkId and DeviceId or LinkId

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

Every Link Association is scoped to exactly one Global Network; the GlobalNetworkId is part of the composite key for the association. Following this link lets you see all other resources (sites, devices, links, transit gateways, etc.) that belong to the same overarching global network.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

The association couples a device to a particular LinkId. Traversing this link shows the underlying circuit or connectivity object that is being attached, along with its provider, bandwidth and cost details.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

The DeviceId in the association identifies the hardware or virtual appliance that terminates the link. Navigating this link reveals the device’s interfaces, status and any other links or connections it participates in.
