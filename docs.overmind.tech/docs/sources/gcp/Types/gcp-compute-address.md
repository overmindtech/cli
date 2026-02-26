---
title: GCP Compute Address
sidebar_label: gcp-compute-address
---

A GCP Compute Address is a reserved, static IP address that can be either regional (tied to a specific region and VPC network) or global (usable by global load-balancing resources). Once reserved, the address can be attached to forwarding rules, virtual machine (VM) instances, Cloud NAT configurations and other networking resources, ensuring its IP does not change even if the underlying resource is recreated. See the official documentation for full details: https://cloud.google.com/compute/docs/ip-addresses/reserve-static-external-ip-address.

**Terrafrom Mappings:**

  * `google_compute_address.name`

## Supported Methods

* `GET`: Get GCP Compute Address by "gcp-compute-address-name"
* `LIST`: List all GCP Compute Address items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-address`](/sources/gcp/Types/gcp-compute-address)

Static addresses rarely reference one another directly, but Overmind may surface links where an address is used as a reference target (for example, when one resource releases and another takes ownership of the same address).

### [`gcp-compute-forwarding-rule`](/sources/gcp/Types/gcp-compute-forwarding-rule)

Regional forwarding rules for Network Load Balancers or protocol forwarding can be configured with a specific static IP. The forwarding rule’s `IPAddress` field points to the Compute Address.

### [`gcp-compute-global-forwarding-rule`](/sources/gcp/Types/gcp-compute-global-forwarding-rule)

Global forwarding rules, used by HTTP(S), SSL, or TCP Proxy load balancers, reference a global static IP address. The global forwarding rule therefore links back to the associated Compute Address.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

A VM instance’s network interface may be assigned a reserved external or internal IP. If an instance uses a static IP, the instance resource contains a link to the corresponding Compute Address.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Internal (private) static addresses are always allocated within a specific VPC network. The Compute Address resource stores the ID of the network from which the IP is taken, creating a link to the Network.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

When you own a public delegated prefix, you can allocate individual static addresses from that range. Each resulting Compute Address records the delegated prefix it belongs to.

### [`gcp-compute-router`](/sources/gcp/Types/gcp-compute-router)

Cloud NAT configurations on a Cloud Router can consume one or more reserved external IP addresses. The router’s NAT config lists the Compute Addresses being used, forming a link.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

For regional internal addresses you must specify the subnetwork (IP range) to allocate from. The Compute Address therefore references, and is linked to, the Subnetwork resource.