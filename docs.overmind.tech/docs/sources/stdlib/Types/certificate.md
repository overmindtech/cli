---
title: Certificate
sidebar_label: certificate
---

A Certificate resource represents an X.509 public-key certificate (typically served during a TLS/SSL handshake) together with any intermediate certificates that form its trust chain. Overmind analyses these certificates to surface risks such as imminent expiry, weak signature algorithms, incorrect key usage flags, or hostnames that do not match the Subject Alternative Names (SANs).  
For the formal specification of X.509 certificates, see RFC 5280 – Internet X.509 Public Key Infrastructure Certificate and Certificate Revocation List (CRL) Profile: https://datatracker.ietf.org/doc/html/rfc5280

## Supported Methods

- ~~`GET`~~
- ~~`LIST`~~
- `SEARCH`: Takes a full certificate, or certificate bundle as input in PEM encoded format
