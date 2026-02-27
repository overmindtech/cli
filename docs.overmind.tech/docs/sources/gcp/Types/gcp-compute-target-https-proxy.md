---
title: GCP Compute Target Https Proxy
sidebar_label: gcp-compute-target-https-proxy
---

A **Target HTTPS Proxy** is a global Google Cloud resource that terminates incoming HTTPS connections at the edge of Google’s network, presents one or more SSL certificates, and then forwards the decrypted requests to the appropriate backend service according to a URL map. In essence, it is the control point that binds SSL certificates, SSL policies, and URL maps together to enable HTTPS traffic on an External HTTP(S) Load Balancer.  
For full details see the official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpsProxies

**Terrafrom Mappings:**

* `google_compute_target_https_proxy.name`

## Supported Methods

* `GET`: Get a gcp-compute-target-https-proxy by its "name"
* `LIST`: List all gcp-compute-target-https-proxy
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-ssl-certificate`](/sources/gcp/Types/gcp-compute-ssl-certificate)

A Target HTTPS Proxy references one or more SSL certificates that it presents to clients during the TLS handshake. Overmind links these certificates so you can track which certificate is in use and assess expiry or misconfiguration risks.

### [`gcp-compute-ssl-policy`](/sources/gcp/Types/gcp-compute-ssl-policy)

An optional SSL policy can be attached to a Target HTTPS Proxy to enforce minimum TLS versions and cipher suites. Overmind exposes this link to highlight the security posture enforced on the proxy.

### [`gcp-compute-url-map`](/sources/gcp/Types/gcp-compute-url-map)

Every Target HTTPS Proxy must point to exactly one URL map, which defines how incoming requests are routed to backend services. Overmind links the URL map so you can trace the full request path and evaluate routing risks before deployment.
