---
title: Hosted Connection
sidebar_label: directconnect-hosted-connection
---

A **Hosted Connection** is an AWS Direct Connect circuit that is provisioned for you by an AWS Direct Connect Delivery Partner on their own network infrastructure and then allocated to your AWS account. It provides a dedicated, layer-2 link that terminates at an AWS Direct Connect location and can be used to create virtual interfaces (VIFs) to access AWS services or your VPCs. Unlike dedicated connections, hosted connections are requested from the partner rather than from AWS directly, and their capacity is limited to 50 Mbps, 100 Mbps, 200 Mbps, 300 Mbps, 400 Mbps or 500 Mbps.  
See the official AWS documentation for full details: https://docs.aws.amazon.com/directconnect/latest/UserGuide/WorkingWithConnections.html#HostedConnections

**Terrafrom Mappings:**

- `aws_dx_hosted_connection.id`

## Supported Methods

- `GET`: Get a Hosted Connection by connection ID
- ~~`LIST`~~
- `SEARCH`: Search Hosted Connections by Interconnect or LAG ID

## Possible Links

### [`directconnect-lag`](/sources/aws/Types/directconnect-lag)

A hosted connection can be delivered over a Link Aggregation Group (LAG). In this case the LAG is the parent resource that physically contains the hosted connection, so the hosted connection links **to** its associated LAG.

### [`directconnect-location`](/sources/aws/Types/directconnect-location)

Every hosted connection terminates at a specific AWS Direct Connect location (for example, a colocation data centre). The hosted connection therefore links **to** the location where its physical port is situated.

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

After a hosted connection becomes available you create one or more virtual interfaces on top of it. These virtual interfaces depend on the hosted connection, so they link **from** the hosted connection.
