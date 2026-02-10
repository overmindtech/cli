---
title: Networkmanager Device
sidebar_label: networkmanager-device
---

An AWS Network Manager Device represents a physical or virtual network appliance (e.g. router, firewall, SD-WAN box, software VPN endpoint) that you register with a Global Network in AWS Network Manager. Once registered, the device becomes a first-class object that can be linked to Sites, Links and Connections, allowing you to model and monitor your entire hybrid network topology in AWS.
For full details see the AWS API reference: https://docs.aws.amazon.com/networkmanager/latest/APIReference/API_Device.html

**Terrafrom Mappings:**

- `aws_networkmanager_device.arn`

## Supported Methods

- `GET`: Get a Networkmanager Device
- ~~`LIST`~~
- `SEARCH`: Search for Networkmanager Devices by GlobalNetworkId, `{GlobalNetworkId|SiteId}` or ARN

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

A device is always created inside a single Global Network. This link shows which Global Network the device belongs to so you can understand its administrative domain.

### [`networkmanager-site`](/sources/aws/Types/networkmanager-site)

Each device is associated with one Site (for example, a particular data centre or branch office). The link reveals the physical location context of the device.

### [`networkmanager-link-association`](/sources/aws/Types/networkmanager-link-association)

A device can have one or more Link Associations that describe the physical or logical circuits (Links) terminating on that device. Following this link surfaces the underlying connectivity for the device.

### [`networkmanager-connection`](/sources/aws/Types/networkmanager-connection)

Connections model the logical relationship between two devices. This link lists all point-to-point or multi-point Connections in which the device participates.

### [`networkmanager-network-resource-relationship`](/sources/aws/Types/networkmanager-network-resource-relationship)

This link captures any additional resource relationships (for example, Transit Gateway attachments or VPNs) that reference the device, providing a holistic view of dependencies and potential blast-radius.
