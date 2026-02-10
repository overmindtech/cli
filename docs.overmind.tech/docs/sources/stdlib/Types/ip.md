---
title: IP Address
sidebar_label: ip
---

An IP address is a numerical label assigned to every device connected to an Internet Protocol network. It uniquely identifies the source and destination of traffic and is used for routing packets across interconnected networks. Overmind treats each IPv4 or IPv6 address that appears in your configuration as a discrete resource, allowing you to map how code, infrastructure and third-party services depend on it and to identify security or availability risks before deployment.  
Official specification documents can be found in the relevant IETF RFCs: IPv4 is defined in RFC 791 and IPv6 in RFC 8200 (see https://www.rfc-editor.org/rfc/rfc791 and https://www.rfc-editor.org/rfc/rfc8200).

## Supported Methods

- `GET`: An ipv4 or ipv6 address
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

DNS records (such as A, AAAA or PTR) map human-readable hostnames to this IP address or resolve the address back to a hostname. Overmind links the `ip` resource to `dns` items whenever the address appears in one of these records so that you can trace how name resolution affects your deployment.

### [`rdap-ip-network`](/sources/stdlib/Types/rdap-ip-network)

Querying the Registration Data Access Protocol (RDAP) for an IP returns information about the allocation block, the organisation that owns it, contact details and abuse mailboxes. Overmind links an `ip` to the corresponding `rdap-ip-network` resource to surface ownership, geolocation and abuse-handling context that may influence compliance or threat-modelling decisions.
