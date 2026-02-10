---
title: Networkmanager Core Network Policy
sidebar_label: networkmanager-core-network-policy
---

An AWS Network Manager Core Network Policy represents the set of declarative rules that describe how traffic may flow within and between the segments of an AWS Cloud WAN core network (for example, how on-premises VPNs, VPCs and Transit Gateways are connected, and which segments are allowed to communicate). Each policy is versioned and attached to a single core network, allowing you to stage, validate and apply changes safely. For further details see the AWS documentation: https://docs.aws.amazon.com/network-manager/latest/cloudwan/cloudwan-policy-operations.html

**Terrafrom Mappings:**

- `aws_networkmanager_core_network_policy.core_network_id`

## Supported Methods

- `GET`: Get a Networkmanager Core Network Policy by Core Network id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Every core network policy is bound to exactly one core network; therefore, Overmind links a `networkmanager-core-network-policy` item back to the corresponding `networkmanager-core-network` to show which core network the policy governs and to make it easier to assess the blast-radius of changes.
