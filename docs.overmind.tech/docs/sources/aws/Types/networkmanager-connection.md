---
title: Networkmanager Connection
sidebar_label: networkmanager-connection
---

An AWS Network Manager Connection represents the logical relationship between two network devices (for example, a branch router and a transit gateway) inside an AWS Global Network. It stores metadata about how the two endpoints are linked, enabling Network Manager to map, monitor and troubleshoot your private WAN from a single view. See the official AWS documentation for full details: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_Connection.html

**Terrafrom Mappings:**

- `aws_networkmanager_connection.arn`

## Supported Methods

- `GET`: Get a Networkmanager Connection
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager Connections by GlobalNetworkId, Device ARN, or Connection ARN

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

Every connection is created within exactly one Global Network. Overmind follows this link to understand which overarching corporate network the connection belongs to and to enumerate all other resources that share the same scope.

### [`networkmanager-link`](/sources/aws/Types/networkmanager-link)

A connection is realised by one or two underlying Links, representing the actual circuits or VPN tunnels that carry traffic. Linking to these allows Overmind to surface characteristics such as bandwidth, provider and health for each side of the connection.

### [`networkmanager-device`](/sources/aws/Types/networkmanager-device)

Each connection terminates on two Devices (the `SourceDeviceId` and `DestinationDeviceId`). From a connection, Overmind can pivot to the involved devices to reveal their configurations, attached links and any downstream dependencies that could be affected by changes to the connection.
