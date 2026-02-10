---
title: Network Firewall TLS Inspection Configuration
sidebar_label: network-firewall-tls-inspection-configuration
---

An AWS Network Firewall TLS Inspection Configuration represents the collection of certificates and related settings that AWS Network Firewall uses to decrypt, inspect and, when appropriate, re-encrypt TLS-encrypted traffic flowing through a firewall. The configuration is referenced by a firewall policy and allows the firewall to analyse traffic that would otherwise be opaque, enabling the detection of threats hidden inside encrypted sessions.  
For full details, see the AWS documentation: https://docs.aws.amazon.com/network-firewall/latest/developerguide/tls-inspection-configuration.html

## Supported Methods

- `GET`: Get a Network Firewall TLS Inspection Configuration by name
- `LIST`: List Network Firewall TLS Inspection Configurations
- `SEARCH`: Search for Network Firewall TLS Inspection Configurations by ARN
