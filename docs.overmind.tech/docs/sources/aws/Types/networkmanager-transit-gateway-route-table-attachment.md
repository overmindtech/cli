---
title: Networkmanager Transit Gateway Route Table Attachment
sidebar_label: networkmanager-transit-gateway-route-table-attachment
---

The Network Manager Transit Gateway Route Table Attachment represents the binding between an AWS Transit Gateway (TGW) route table and an AWS Cloud WAN (Network Manager Core Network) segment. Creating this attachment allows routes that exist in the TGW route table to be advertised into the Cloud WAN segment and, conversely, permits segment routes to be propagated to the TGW. In effect, it provides a controlled integration point between an existing TGW-based topology and a Cloud WAN fabric.  
Official API documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_CreateTransitGatewayRouteTableAttachment.html

**Terrafrom Mappings:**

- `aws_networkmanager_transit_gateway_route_table_attachment.id`

## Supported Methods

- `GET`: Get a Networkmanager Transit Gateway Route Table Attachment by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Every Transit Gateway Route Table Attachment is created inside a specific Core Network and targets one of its segments. Therefore, the attachment is a child resource of the Core Network and inherits its administrative domain and policy constraints.

### [`networkmanager-transit-gateway-peering`](/sources/aws/Types/networkmanager-transit-gateway-peering)

Before a TGW route table can be attached, a Transit Gateway Peering must exist between the TGW and the Core Network. The attachment references that peering to determine the underlay connection over which route exchange will occur.
