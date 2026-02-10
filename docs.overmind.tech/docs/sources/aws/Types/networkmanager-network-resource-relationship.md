---
title: Networkmanager Network Resource Relationships
sidebar_label: networkmanager-network-resource-relationship
---

Represents an association between two AWS Network Manager resources within a single Global Network. A Network Resource Relationship records how different components—such as devices, links, connections and Direct Connect objects—are connected, enabling topology visualisation and impact analysis. Each relationship object identifies a **source resource**, a **destination resource**, and the **type of relationship** (for example `CONNECTED_TO` or `CHILD_OF`).  
For full field-level details see the AWS API reference: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_NetworkResourceRelationship.html

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager NetworkResourceRelationships by GlobalNetworkId

## Possible Links

### [`networkmanager-connection`](/sources/aws/Types/networkmanager-connection)

A Network Manager connection (for example a VPN or Transit Gateway attachment) can appear as either the **source** or **destination** in a relationship, indicating that it is logically connected to another resource—most commonly a site, device or Direct Connect virtual interface.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

Devices (routers, firewalls or SD-WAN appliances) are frequently linked to links and connections. When a device participates in a relationship, the record shows which link it uses or which connection terminates on the device.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

A link represents physical or logical connectivity (for example an MPLS circuit). Relationships illustrate which device, site or Direct Connect virtual interface is using, or is reached through, a given link.

### [`networkmanager-site`](/sources/aws/Types/networkmanager-site)

Site resources group devices and links. Relationships referencing a site capture a **CHILD_OF** type association, showing that a particular device or link belongs to, or is located within, the site.

### [`directconnect-connection`](/sources/aws/Types/directconnect-connection)

Direct Connect connections are mapped into the global network; relationships show how a Direct Connect line is attached to a Network Manager link or gateway, providing visibility of dedicated connectivity paths.

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

When a Direct Connect gateway is part of a global network, relationships identify which connections or virtual interfaces are routed through the gateway, enabling you to trace traffic flows.

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

Virtual interfaces (private, public or transit) may be related to Direct Connect connections, gateways or Network Manager links. The relationship clarifies which physical connection a VIF is presented on and how it integrates with the wider network.
