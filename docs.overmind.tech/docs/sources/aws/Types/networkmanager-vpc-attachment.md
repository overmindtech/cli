---
title: Networkmanager VPC Attachment
sidebar_label: networkmanager-vpc-attachment
---

A Network Manager VPC attachment represents the logical link between an Amazon Virtual Private Cloud (VPC) and an AWS Cloud WAN / Network Manager **core network**. By creating an attachment you allow the sub-nets inside the VPC to participate in the global routing domain managed by Network Manager, making it possible for traffic to reach other VPCs, on-premises networks, or SD-WAN devices that are also attached to the same core network.
For a detailed explanation of the resource and its properties, see the [official AWS documentation](https://docs.aws.amazon.com/vpc/latest/cloudwan/what-is-cloudwan.html).

**Terrafrom Mappings:**

- `aws_networkmanager_vpc_attachment.id`

## Supported Methods

- `GET`: Get a Networkmanager VPC Attachment by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Every VPC attachment is created inside a specific core network and inherits its routing policies. The `core_network_id` field on the attachment identifies that parent, so Overmind can follow this link to reveal the wider network fabric that the VPC will join.
