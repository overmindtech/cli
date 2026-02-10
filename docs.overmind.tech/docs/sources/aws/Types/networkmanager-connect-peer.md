---
title: Networkmanager Connect Peer
sidebar_label: networkmanager-connect-peer
---

An AWS Network Manager **Connect Peer** represents one end of a GRE tunnel that is established over a Network Manager _Connect attachment_ (for example, between an AWS Transit Gateway/Cloud WAN core network and an external router).  
The peer stores the tunnel’s **inside and outside IP addresses**, BGP configuration (peer ASN, BGP addresses and keys), the subnet in which the tunnel terminates, and the current operational state. Creating the peer is the final step that brings a Connect attachment into service, enabling traffic to flow between AWS and on-premises or third-party networks.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_ConnectPeer.html

**Terrafrom Mappings:**

- `aws_networkmanager_connect_peer.id`

## Supported Methods

- `GET`: Get a Networkmanager Connect Peer by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

A Connect peer ultimately belongs to a core network; through its parent Connect attachment it is associated with a specific core network ID, so the peer can be traced back to the Cloud WAN or Transit Gateway core it serves.

### [`networkmanager-connect-attachment`](/sources/aws/Types/networkmanager-connect-attachment)

Each Connect peer is created **within** a single Connect attachment. This link identifies the attachment that houses the peer and through which the GRE tunnel is terminated.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

The peer exposes both _inside_ and _outside_ tunnel IP addresses. These addresses are modelled as IP resources and linked so you can see which IPs are consumed by the peer.

### [`rdap-asn`](/sources/stdlib/Types/rdap-asn)

When BGP is enabled the peer records the remote BGP ASN. Overmind links that ASN so you can quickly inspect public registration information for the autonomous system you are peering with.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

The peer must be associated with a specific subnet that contains the tunnel’s AWS endpoint. Linking to the EC2 subnet shows the precise network segment in which the peer resides, helping to check routing and security settings.
