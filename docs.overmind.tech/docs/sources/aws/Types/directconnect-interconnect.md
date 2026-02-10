---
title: Interconnect
sidebar_label: directconnect-interconnect
---

An AWS Direct Connect **Interconnect** is a high-capacity physical Ethernet link (10 Gbps or 100 Gbps) between an AWS Direct Connect location and the network of an approved network service provider. The provider uses the interconnect to carve out and allocate Hosted Connections or Hosted Virtual Interfaces for individual customer accounts, allowing many end-users to share the same physical infrastructure while maintaining logical separation and security. In Overmind, the **directconnect-interconnect** type lets you surface configuration details (such as bandwidth, location, and operational state) and map its relationships to other Direct Connect resources so you can spot mis-configuration or single-point-of-failure risks before deployment.  
For authoritative information see the AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/WorkingWithInterconnects.html

## Supported Methods

- `GET`: Get a Interconnect by InterconnectId
- `LIST`: List all Interconnects
- `SEARCH`: Search Interconnects by ARN

## Possible Links

### [`directconnect-hosted-connection`](/sources/aws/Types/directconnect-hosted-connection)

Hosted connections are provisioned on top of an Interconnect. Each hosted connection link points back to the parent Interconnect that physically carries its traffic.

### [`directconnect-lag`](/sources/aws/Types/directconnect-lag)

LAGs (Link Aggregation Groups) created on an Interconnect combine multiple physical ports of that Interconnect into a single logical interface, increasing bandwidth and providing redundancy.

### [`directconnect-location`](/sources/aws/Types/directconnect-location)

Every Interconnect terminates at a specific Direct Connect location such as an AWS-aligned colocation facility; this link shows where the Interconnect is physically hosted.
