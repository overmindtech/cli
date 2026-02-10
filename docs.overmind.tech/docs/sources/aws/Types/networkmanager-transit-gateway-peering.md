---
title: Networkmanager Transit Gateway Peering
sidebar_label: networkmanager-transit-gateway-peering
---

An AWS Network Manager Transit Gateway Peering represents a peering attachment between an AWS Cloud WAN _core network_ and an existing AWS Transit Gateway (TGW). Creating this peering allows traffic to flow transparently between VPCs or on-premises networks connected to the Transit Gateway and the segments that make up the Cloud WAN core network, extending the reach of both fabrics without the need for additional VPNs or direct-connect links.
For more information, see the [AWS documentation](https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_TransitGatewayPeering.html).

**Terrafrom Mappings:**

- `aws_networkmanager_transit_gateway_peering.id`

## Supported Methods

- `GET`: Get a Networkmanager Transit Gateway Peering by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Every Transit Gateway Peering is created **within** a specific Cloud WAN core network; the core network is the logical container that owns the peering attachment. Consequently, querying a `networkmanager-core-network` item allows you to enumerate or drill down to its associated Transit Gateway Peerings, and conversely, each Transit Gateway Peering stores the identifier of the core network it belongs to.
