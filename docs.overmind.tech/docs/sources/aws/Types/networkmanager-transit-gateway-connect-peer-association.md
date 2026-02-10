---
title: Networkmanager Transit Gateway Connect Peer Association
sidebar_label: networkmanager-transit-gateway-connect-peer-association
---

A Network Manager Transit Gateway Connect Peer Association represents the connection between an AWS Transit Gateway Connect peer (a GRE tunnel endpoint created as part of a Transit Gateway Connect attachment) and a site that you have modelled inside AWS Network Manager.  
The object records which Global Network the peer belongs to and, optionally, which on-premises device and physical/virtual link it should be mapped to. Maintaining this mapping allows Network Manager to draw accurate topology diagrams and to include the GRE tunnel in route analytics, performance monitoring, and policy assessments.

Official documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_TransitGatewayConnectPeerAssociation.html

## Supported Methods

- `GET`: Get a Networkmanager Transit Gateway Connect Peer Association by id
- `LIST`: List all Networkmanager Transit Gateway Connect Peer Associations
- `SEARCH`: Search for Networkmanager Transit Gateway Connect Peer Associations by GlobalNetworkId

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

Every Transit Gateway Connect Peer Association is scoped to a single Global Network. The `GlobalNetworkId` on the association points to the corresponding `networkmanager-global-network` item, indicating which overall corporate network the peer is part of.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

The association can specify a `DeviceId` to indicate the on-premises or edge device (for example, a customer router or firewall) that terminates the GRE tunnel. Linking to the `networkmanager-device` item shows where the peer logically lands in your topology.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

If the Connect peer is tied to a particular circuit, VLAN, or VPN link at the site, the association includes a `LinkId`. This links the peer to a `networkmanager-link` item, allowing you to trace the physical or logical connectivity that underpins the GRE tunnel.
