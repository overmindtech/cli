---
title: RDAP Domain
sidebar_label: rdap-domain
---

An RDAP Domain record represents the authoritative registration data for a domain name as returned by the Registration Data Access Protocol (RDAP). The record contains information such as the registrar, registrant and administrative contacts, name-servers, status flags (e.g. `clientTransferProhibited`), and important lifecycle dates (creation, expiry, last update). In Overmind the resource lets you inspect this registration data and understand how a domain fits into the rest of your deployment before any changes are made.  
Official RDAP specification: https://www.rfc-editor.org/rfc/rfc9082

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- `SEARCH`: Search for a domain record by the domain name e.g. "www.google.com"

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

The name portion of the RDAP domain (e.g. `example.com`) will typically have authoritative DNS records such as `A`, `AAAA`, `MX`, etc. Overmind links the RDAP Domain to those `dns` items so that you can trace from the registration layer straight through to the operational zone file that will actually be served.

### [`rdap-nameserver`](/sources/stdlib/Types/rdap-nameserver)

An RDAP Domain record contains a list of host objects (name-servers) delegated for the zone. Each of those host objects is represented as an `rdap-nameserver` item. The link allows you to drill into the registration data for each individual name-server.

### [`rdap-entity`](/sources/stdlib/Types/rdap-entity)

Entities in RDAP describe people or organisations such as the registrant, administrative contact, or registrar. Overmind links the RDAP Domain to every referenced `rdap-entity` so that you can view contact details, roles and other domains controlled by the same party.

### [`rdap-ip-network`](/sources/stdlib/Types/rdap-ip-network)

If the RDAP Domain record (or any of its linked name-servers) includes embedded references to address space—commonly via `v4network` or `v6network` objects—Overmind exposes those as `rdap-ip-network` items. This lets you see which blocks of IP addresses are directly associated with the domain and whether they overlap with other infrastructure you manage.
