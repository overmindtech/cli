---
title: RDAP Entity
sidebar_label: rdap-entity
---

An RDAP (Registration Data Access Protocol) Entity resource represents a single contact object – either a person, organisation or role – that appears in the registration data held by Regional Internet Registries (RIRs) and other RDAP servers. It typically contains identifying information such as names, postal addresses, e-mail addresses, telephone numbers and public identifiers, and is referenced by other RDAP objects (e.g. ASNs, IPv4/IPv6 prefix ranges and domain names) as their administrative, technical or abuse contact.

The formal structure and semantics of an RDAP Entity are defined in RFC 9083, section 5.1 (https://www.rfc-editor.org/rfc/rfc9083#section-5.1).

## Supported Methods

- `GET`: Get an entity by its handle. This method is discouraged as it's not reliable since entity bootstrapping isn't comprehensive
- ~~`LIST`~~
- `SEARCH`: Search for an entity by its URL e.g. https://rdap.apnic.net/entity/AIC3-AP

## Possible Links

### [`rdap-asn`](/sources/stdlib/Types/rdap-asn)

An ASN record can reference one or more RDAP Entities as its registrant, administrative or technical contacts. Overmind links the rdap-asn resource to the corresponding rdap-entity resources so that you can see who is responsible for a particular Autonomous System and assess any associated risk or exposure stemming from those contacts.
