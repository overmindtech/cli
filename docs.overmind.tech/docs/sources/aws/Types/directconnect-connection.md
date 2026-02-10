---
title: Connection
sidebar_label: directconnect-connection
---

An AWS Direct Connect Connection represents a single dedicated network circuit between your on-premises environment (or colocation facility) and an AWS Direct Connect location. By provisioning a connection you obtain a physical 1 Gbps, 10 Gbps or 100 Gbps port on an AWS router, through which you can create one or more virtual interfaces to reach AWS services or your VPCs. A connection is the fundamental building-block for achieving consistent, low-latency private connectivity into AWS, bypassing the public Internet and allowing you to commit to specific bandwidth and service-level requirements. See the official AWS documentation for further details: https://docs.aws.amazon.com/directconnect/latest/UserGuide/WorkingWithConnections.html

**Terrafrom Mappings:**

- `aws_dx_connection.id`

## Supported Methods

- `GET`: Get a connection by ID
- `LIST`: List all connections
- `SEARCH`: Search connection by ARN

## Possible Links

### [`directconnect-lag`](/sources/aws/Types/directconnect-lag)

A Link Aggregation Group (LAG) can aggregate one or more individual connections into a single managed logical interface. A connection may belong to a LAG, and conversely a LAG lists each underlying connection that forms part of the group.

### [`directconnect-location`](/sources/aws/Types/directconnect-location)

Every connection is terminated at a specific Direct Connect location (e.g. an Equinix or Digital Realty data centre). The connection resource references its chosen location to indicate where the physical port is installed.

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

Virtual interfaces (public, private or transit) are configured on top of a connection to carry customer traffic. Each virtual interface is associated with exactly one connection (or LAG), while a single connection can host multiple virtual interfaces for different routing purposes.
