---
title: DNS Entry
sidebar_label: dns
---

The Domain Name System (DNS) translates human-readable names into machine-usable information. A DNS _A_ record maps a hostname to an IPv4 address, while an _AAAA_ record maps it to an IPv6 address. By querying these records, Overmind can reveal the infrastructure that a name ultimately points to, allowing you to spot configuration mistakes, dangling records, or unexpected dependencies before you deploy.  
Reference documentation: RFC 1034 & RFC 1035 – Domain Names – Concepts and Facilities / Implementation and Specification (https://www.rfc-editor.org/rfc/rfc1034 and https://www.rfc-editor.org/rfc/rfc1035)

## Supported Methods

- `GET`: A DNS A or AAAA entry to look up
- ~~`LIST`~~
- `SEARCH`: A DNS name (or IP for reverse DNS), this will perform a recursive search and return all results. It is recommended that you always use the SEARCH method

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

If the queried record is a CNAME, MX, NS or contains additional glue, Overmind follows those pointers and links the resulting records back as further `dns` items for deeper traversal.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

An A or AAAA record resolves to one or more IP addresses; each discovered address is linked as an `ip` item so their ownership, location and associated services can be examined.

### [`rdap-domain`](/sources/stdlib/Types/rdap-domain)

The second-level or higher-level domain extracted from the DNS name is linked to its corresponding `rdap-domain` item, giving visibility into registrar, registrant and name-server information that may present additional risk factors.
