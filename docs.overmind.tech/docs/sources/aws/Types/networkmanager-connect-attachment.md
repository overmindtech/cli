---
title: Networkmanager Connect Attachment
sidebar_label: networkmanager-connect-attachment
---

A Network Manager Connect Attachment represents the logical connection used to link a third-party SD-WAN, on-premises router or other non-AWS network appliance to an AWS Cloud WAN core network. It enables you to extend a core network beyond AWS, transporting traffic through GRE tunnels that are established and maintained by a subsequently created Connect Peer.  
For full details see the AWS documentation: https://docs.aws.amazon.com/network-manager/latest/cloudwan/cloudwan-network-attachments.html#cloudwan-attachment-connect

**Terrafrom Mappings:**

- `aws_networkmanager_core_network.id`

## Supported Methods

- `GET`:

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Every Connect Attachment is created inside a specific Cloud WAN core network, referenced by its `CoreNetworkId`. Consequently, Overmind links a connect attachment back to the corresponding `networkmanager-core-network` so that you can trace how external connectivity feeds into, and potentially affects, the wider Cloud WAN topology.
