---
title: Networkmanager Site To Site Vpn Attachment
sidebar_label: networkmanager-site-to-site-vpn-attachment
---

A Network Manager Site-to-Site VPN attachment represents the connection of an AWS Site-to-Site VPN to an AWS Cloud WAN / Network Manager core network. By creating this attachment you allow traffic from a remote on-premises site, carried over an IPsec VPN tunnel, to be routed through the core network alongside other AWS and on-premises connections.
Further information can be found in the [official AWS documentation](https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_SiteToSiteVpnAttachment.html).

**Terrafrom Mappings:**

- `aws_networkmanager_site_to_site_vpn_attachment.id`

## Supported Methods

- `GET`: Get a Networkmanager Site To Site Vpn Attachment by id
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`networkmanager-core-network`](/sources/aws/Types/networkmanager-core-network)

Each Site-to-Site VPN attachment is created inside a single core network, so the attachment item is linked to the `networkmanager-core-network` that owns it.
