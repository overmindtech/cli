---
title: Networkmanager Connect Peer Association
sidebar_label: networkmanager-connect-peer-association
---

An AWS Network Manager **Connect Peer Association** records the relationship between a Transit Gateway Connect peer and the on-premises device and link through which that peer reaches the AWS global network. It lets you see which Connect peers are presently attached to which devices and links inside a particular Global Network, and in which state the attachment currently is (for example, _pending_ or _available_).  
For full API details, refer to the official AWS documentation: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_ConnectPeerAssociation.html

## Supported Methods

- `GET`: Get a Networkmanager Connect Peer Association
- `LIST`: List all Networkmanager Connect Peer Associations
- `SEARCH`: Search for Networkmanager ConnectPeerAssociations by GlobalNetworkId

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

The association is scoped to a single Global Network; every Connect Peer Association includes the `GlobalNetworkId` that ties it back to this parent resource.

### [`networkmanager-connect-peer`](/sources/aws/Types/networkmanager-connect-peer)

The association identifies the specific Connect Peer (`ConnectPeerId`) whose attachment details are being tracked.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

If the Connect peer terminates on a particular on-premises or edge device, the association includes the `DeviceId`, linking it to this device resource.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

Where applicable, the association also records the `LinkId`, showing which physical or logical link is being used by the Connect peer to reach AWS.
