---
title: GCP Compute Target Https Proxy
sidebar_label: gcp-compute-target-https-proxy
---

A **Target HTTPS Proxy** is a global Google Cloud resource that terminates incoming HTTPS traffic and forwards the decrypted requests to the appropriate backend service according to a referenced URL map. It is a central component of the External HTTP(S) Load Balancer, holding one or more SSL certificates that are presented to clients during the TLS handshake and optionally enforcing an SSL policy that dictates the allowed protocol versions and cipher suites.
Official documentation: https://docs.cloud.google.com/sdk/gcloud/reference/compute/target-https-proxies

**Terrafrom Mappings:**

- `google_compute_target_https_proxy.name`

## Supported Methods

- `GET`: Get a gcp-compute-target-https-proxy by its "name"
- `LIST`: List all gcp-compute-target-https-proxy
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-ssl-certificate`](/sources/gcp/Types/gcp-compute-ssl-certificate)

The proxy references one or more SSL certificates that are served to clients when they initiate an HTTPS connection. These certificates are specified in the `ssl_certificates` field of the target HTTPS proxy.

### [`gcp-compute-ssl-policy`](/sources/gcp/Types/gcp-compute-ssl-policy)

An optional SSL policy can be attached to the proxy to control minimum TLS versions, allowed cipher suites, and other security settings. The policy is linked through the `ssl_policy` attribute.

### [`gcp-compute-url-map`](/sources/gcp/Types/gcp-compute-url-map)

Each target HTTPS proxy must reference exactly one URL map, which defines the routing rules that determine which backend service receives each request after SSL/TLS termination.
