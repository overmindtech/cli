---
title: Link Aggregation Group
sidebar_label: directconnect-lag
---

An AWS Direct Connect **Link Aggregation Group (LAG)** allows you to combine multiple physical Direct Connect connections into a single, logical interface. Doing so simplifies management, provides higher aggregate bandwidth and offers built-in resiliency: if one underlying connection goes down, traffic is automatically redistributed across the remaining links. Each LAG behaves as a single port on the AWS side while still exposing the individual connections (with their own light-levels and alarms) for troubleshooting.  
Official AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/lag.html

**Terrafrom Mappings:**

- `aws_dx_lag.id`

## Supported Methods

- `GET`: Get a Link Aggregation Group by ID
- `LIST`: List all Link Aggregation Groups
- `SEARCH`: Search Link Aggregation Group by ARN

## Possible Links

### [`directconnect-connection`](/sources/aws/Types/directconnect-connection)

A LAG is essentially a collection of Direct Connect connections. Each linked `directconnect-connection` represents one of the physical ports that has been bundled into the LAG.

### [`directconnect-hosted-connection`](/sources/aws/Types/directconnect-hosted-connection)

Hosted connections can also be associated with a LAG. Overmind links these `directconnect-hosted-connection` resources to show which hosted (customer-provisioned) circuits are aggregated under the same LAG.

### [`directconnect-location`](/sources/aws/Types/directconnect-location)

Every LAG is created at a specific AWS Direct Connect location (data centre or colocation facility). The `directconnect-location` link identifies the physical site where the LAG’s constituent connections terminate.
