---
title: Autonomous System Number (ASN)
sidebar_label: rdap-asn
---

An Autonomous System Number (ASN) is a unique 16- or 32-bit identifier assigned to an Autonomous System so that it can participate in Border Gateway Protocol (BGP) routing on the public Internet. Using the Registration Data Access Protocol (RDAP), you can query an ASN to obtain registration details such as the holder, allocation status, and associated contacts. For the formal specification of RDAP responses for ASNs, see [RFC 9083: Registration Data Access Protocol (RDAP)](https://datatracker.ietf.org/doc/html/rfc9083).

## Supported Methods

- `GET`: Get an ASN by handle i.e. "AS15169"
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`rdap-entity`](/sources/stdlib/Types/rdap-entity)

An ASN RDAP record frequently contains an `entities` array. Each item in that array is an RDAP Entity object representing the organisation or individual responsible for the ASN (registrant, administrative contact, technical contact, etc.). Overmind therefore links an `rdap-asn` resource to one or more `rdap-entity` resources so that you can inspect the people or organisations behind a particular network.
