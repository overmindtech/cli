---
title: GCP Compute External Vpn Gateway
sidebar_label: gcp-compute-external-vpn-gateway
---

A **Compute External VPN Gateway** is a Google Cloud resource that represents a customer-managed VPN appliance that resides outside of Google’s network (for example, in an on-premises data centre or another cloud). By defining one or more external interface IP addresses and an associated redundancy type, it tells Cloud VPN (HA VPN or Classic VPN) where to terminate its tunnels. In other words, the resource is the “remote end” of a Cloud VPN connection, allowing Google Cloud to establish secure IPSec tunnels to external infrastructure.  
For further details, see the official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/externalVpnGateways

**Terrafrom Mappings:**

* `google_compute_external_vpn_gateway.name`

## Supported Methods

* `GET`: Get a gcp-compute-external-vpn-gateway by its "name"
* `LIST`: List all gcp-compute-external-vpn-gateway
* ~~`SEARCH`~~
