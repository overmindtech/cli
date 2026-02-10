---
title: RDAP Nameserver
sidebar_label: rdap-nameserver
---

The Registration Data Access Protocol (RDAP) is the modern, machine-readable replacement for the old WHOIS service.  
An **RDAP ­nameserver resource** represents the authoritative information that a Top-Level Domain (TLD) registry publishes about a particular DNS nameserver. By querying this endpoint you can discover, for example, the registrar that manages the server, its associated IP addresses, its status with the registry and any abuse or support contacts.  
For details of the protocol and the structure of a nameserver response, see the IETF specification: https://datatracker.ietf.org/doc/html/rfc7483#section-5.5.

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- `SEARCH`: Search for the RDAP entry for a nameserver by its full URL e.g. "https://rdap.verisign.com/com/v1/nameserver/NS4.GOOGLE.COM"

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

A nameserver appears in DNS NS records. Overmind links the RDAP nameserver object to the corresponding `dns` item so that you can see which zones delegate to this server and whether those zones are also in your inventory.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

The RDAP response normally includes the A and/or AAAA records for the nameserver. These addresses are represented as `ip` items, allowing you to trace from the logical nameserver to the concrete IP resources that sit behind it.

### [`rdap-entity`](/sources/stdlib/Types/rdap-entity)

Each nameserver RDAP document references one or more entities (registrar, registrant, technical contact, abuse contact, etc.). These are captured as separate `rdap-entity` items and linked so you can quickly identify who is responsible for the server and how to contact them.
