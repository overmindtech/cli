---
title: Networkmanager Transit Gateway Registrations
sidebar_label: networkmanager-transit-gateway-registration
---

A Network Manager Transit Gateway Registration represents the association of an AWS Transit Gateway with an AWS Network Manager Global Network. By registering a Transit Gateway, you enable Network Manager to map its attachments, monitor routing changes and performance, and include the gateway in your overall network topology visualisation. For more information, see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/tgw/register-transit-gateway.html

## Supported Methods

- `GET`: Get a Networkmanager Transit Gateway Registrations
- `LIST`: List all Networkmanager Transit Gateway Registrations
- `SEARCH`: Search for Networkmanager Transit Gateway Registrations by GlobalNetworkId

## Possible Links

### [`networkmanager-global-network`](/sources/aws/Types/networkmanager-global-network)

A Transit Gateway registration is always scoped to, and therefore linked with, a single Network Manager Global Network. This link indicates the parent Global Network that owns the registration, allowing Overmind to traverse from the high-level network to the individual Transit Gateway associations.
