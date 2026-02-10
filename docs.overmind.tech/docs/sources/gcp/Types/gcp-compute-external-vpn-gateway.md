---
title: GCP Compute External Vpn Gateway
sidebar_label: gcp-compute-external-vpn-gateway
---

A GCP Compute External VPN Gateway represents a VPN gateway device that resides outside of Google Cloud—typically an on-premises firewall, router or a third-party cloud appliance. In High-Availability VPN (HA VPN) configurations it is used to describe the peer gateway so that Cloud Router and HA VPN tunnels can be created and managed declaratively. Each external gateway resource records the device’s public IP addresses and routing style, allowing Google Cloud to treat the remote endpoint as a first-class object and to validate or reference it from other VPN and network resources.
For full details, see the official Google documentation: https://cloud.google.com/sdk/gcloud/reference/compute/external-vpn-gateways

**Terrafrom Mappings:**

- `google_compute_external_vpn_gateway.name`

## Supported Methods

- `GET`: Get a gcp-compute-external-vpn-gateway by its "name"
- `LIST`: List all gcp-compute-external-vpn-gateway
- ~~`SEARCH`~~
