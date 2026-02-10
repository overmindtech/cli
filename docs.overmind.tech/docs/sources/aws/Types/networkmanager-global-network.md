---
title: Network Manager Global Network
sidebar_label: networkmanager-global-network
---

An AWS Network Manager Global Network is the top-level container that represents your organisation’s private global network within AWS. It groups together sites, on-premises devices, AWS Transit Gateways, and the connections between them, allowing you to view and manage the entire topology from a single place. You must create a global network before you can register any resources with Network Manager.  
Official documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_GlobalNetwork.html

**Terrafrom Mappings:**

- `aws_networkmanager_global_network.arn`

## Supported Methods

- `GET`: Get a global network by id
- `LIST`: List all global networks
- `SEARCH`: Search for a global network by ARN

## Possible Links

### [`networkmanager-site`](/sources/aws/Types/networkmanager-site)

A Site is created inside a Global Network. Each `networkmanager-site` record therefore links back to the Global Network that owns it.

### [`networkmanager-transit-gateway-registration`](/sources/aws/Types/networkmanager-transit-gateway-registration)

Transit Gateways must be registered with a specific Global Network before they can be visualised or managed by Network Manager. These registration objects reference the parent Global Network.

### [`networkmanager-connect-peer-association`](/sources/aws/Types/networkmanager-connect-peer-association)

A Connect Peer Association represents the attachment of a Connect peer to a Global Network. The association record points to the Global Network in which the peer is enrolled.

### [`networkmanager-transit-gateway-connect-peer-association`](/sources/aws/Types/networkmanager-transit-gateway-connect-peer-association)

Similar to the above, but for Transit Gateway Connect peers. The association is made within the scope of a single Global Network.

### [`networkmanager-network-resource-relationship`](/sources/aws/Types/networkmanager-network-resource-relationship)

This type models relationships between any two resources (devices, links, TGWs, etc.) that are part of the same Global Network. Each relationship object is tied to the Global Network it belongs to.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

Links represent the physical or logical connections at a Site and, by extension, sit within the Global Network that the Site is part of.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

Devices (routers, switches, firewalls, etc.) are registered to Sites, and consequently to the parent Global Network. Each device record references its Global Network identifier.

### [`networkmanager-connection`](/sources/aws/Types/networkmanager-connection)

Connections join two Devices over one or more Links inside a Global Network. The connection object therefore includes the Global Network ID to denote its scope.
