---
title: Direct Connect Gateway Attachment
sidebar_label: directconnect-direct-connect-gateway-attachment
---

An AWS Direct Connect **gateway attachment** represents the binding between a Direct Connect Gateway and a Virtual Interface (VIF). When the attachment is in the `attached` state, traffic that reaches the VIF can be routed to any VPCs or on-premises networks that are associated with the gateway, even across accounts or Regions.  
For a full description of the concept, states, and quotas involved, see the AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/direct-connect-gateways.html#dx-gateway-attachments

## Supported Methods

- `GET`: Get a direct connect gateway attachment by DirectConnectGatewayId/VirtualInterfaceId
- ~~`LIST`~~
- `SEARCH`: Search direct connect gateway attachments for given VirtualInterfaceId

## Possible Links

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

Each gateway attachment belongs to exactly one Direct Connect Gateway. Overmind links the attachment back to its parent gateway so you can see every VIF that is currently associated with that gateway.

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

The attachment is also linked to the Virtual Interface that is being attached. This lets you trace which VIFs are connected to which gateways and, in turn, to the networks that sit behind those gateways.
