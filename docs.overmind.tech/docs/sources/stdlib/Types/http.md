---
title: HTTP Endpoint
sidebar_label: http
---

An HTTP Endpoint represents a reachable URL that Overmind can interrogate in order to discover configuration or security issues before deployment. By performing lightweight `HEAD` or `GET` requests, Overmind determines the availability, response headers, redirects, and TLS configuration (if the endpoint is served over HTTPS). This allows you to spot problems such as broken links, unexpected redirections, missing security headers, or invalid certificates early in the pipeline.  
For more background on how HTTP endpoints are conventionally exposed and managed on the internet, refer to the W3C documentation on HTTP semantics: https://www.w3.org/Protocols/ (external).

## Supported Methods

- `GET`: A HTTP endpoint to run a `HEAD` request against
- ~~`LIST`~~
- `SEARCH`: A HTTP URL to search for. Query parameters and fragments will be stripped from the URL before processing.

## Possible Links

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

The hostname or FQDN of the HTTP endpoint ultimately resolves to one or more IP addresses. Overmind records these addresses to understand network-level reachability and to cross-reference them with firewall, VPC or load-balancer configurations.

### [`dns`](/sources/stdlib/Types/dns)

Before an HTTP request can be made, the client performs a DNS lookup. Overmind connects the endpoint to its corresponding DNS records (A, AAAA, CNAME, etc.) so you can see how changes in DNS zone files might affect the endpoint’s availability.

### [`certificate`](/sources/gcp/Types/gcp-compute-ssl-certificate)

If the endpoint is accessed over HTTPS, the server presents an X.509 certificate. Overmind links the endpoint to the certificate resource it observes during the TLS handshake, enabling validation of expiry dates, issuer trust chains, and key strengths.

### [`http`](/sources/stdlib/Types/http)

HTTP endpoints often redirect to, embed, or call other HTTP endpoints (for example via 3xx redirects or links in HTML/JSON responses). Overmind establishes links between them so you can trace dependencies, spot redirect loops, and ensure downstream endpoints meet your security standards.
