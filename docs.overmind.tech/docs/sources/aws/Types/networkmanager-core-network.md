---
title: Networkmanager Core Network
sidebar_label: networkmanager-core-network
---

An AWS Network Manager **core network** represents the logical, centrally-managed backbone created by AWS Cloud WAN. It defines the global routing fabric, network segments, and edge locations that connect your AWS Regions and on-premises sites. Once a core network is in place you can attach VPCs, VPNs, Direct Connects and third-party SD-WAN devices, and let Cloud WAN automatically propagate routes between them according to the policy you supply.
For further details see the [official documentation](https://docs.aws.amazon.com/vpc/latest/cloudwan/what-is-cloudwan.html).

**Terrafrom Mappings:**

- `aws_networkmanager_core_network.id`

## Supported Methods

- `GET`: Get a Networkmanager Core Network by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network-policy`](/sources/aws/Types/networkmanager-core-network-policy)

Every core network is governed by a **core network policy** that declares its segments, attachment permissions, and routing intent. Overmind links a `networkmanager-core-network` to its current `networkmanager-core-network-policy` so that you can inspect or diff the policy that is actively controlling the network.

### [`networkmanager-connect-peer`](/sources/aws/Types/networkmanager-connect-peer)

A **Connect peer** represents a GRE/BGP session that terminates on a Connect attachment belonging to a core network. Overmind exposes this link to show which Connect peers (and therefore which on-premises routers) are logically attached to the given core network.
