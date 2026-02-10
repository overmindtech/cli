---
title: RDAP IP Network
sidebar_label: rdap-ip-network
---

An **RDAP IP Network** represents a block of IPv4 or IPv6 address space as returned by the Registration Data Access Protocol (RDAP). Overmind queries the authoritative RDAP service for a supplied address or prefix and surfaces the resulting network object, revealing who owns the range, the exact start- and end-addresses, its allocation status (allocated, assigned, reserved, etc.), and any policy or abuse information attached to it. Seeing this data in advance helps you verify that the addresses your deployment will use are valid and not bogon, reserved, or owned by an unexpected party.
The RDAP specification for IP networks is defined in [RFC 9083 – Registration Data Access Protocol (RDAP): Query Format](https://datatracker.ietf.org/doc/html/rfc9083).

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- `SEARCH`: Search for the most specific network that contains the specified IP or CIDR

## Possible Links

### [`rdap-entity`](/sources/stdlib/Types/rdap-entity)

An RDAP network record contains an `entities` array referencing the people, organisations, and roles (registrant, technical, abuse, etc.) responsible for the address space. Overmind links each of these references to its corresponding `rdap-entity` item, letting you inspect contact details and responsibility assignments related to the network.
