---
title: GCP Compute Ssl Certificate
sidebar_label: gcp-compute-ssl-certificate
---

A GCP Compute SSL Certificate is a regional resource that stores the public and private key material required to terminate TLS for Google Cloud load balancers and proxy targets. Once created, the certificate can be attached to target HTTPS proxies (for external HTTP(S) Load Balancing) or target SSL proxies (for SSL Proxy Load Balancing) so that incoming connections can be securely encrypted in transit. Certificate data is provided by the user (self-managed) and can later be rotated or deleted as required.  
For full details see the Google Cloud documentation: https://cloud.google.com/compute/docs/reference/rest/v1/sslCertificates

**Terrafrom Mappings:**

- `google_compute_ssl_certificate.name`

## Supported Methods

- `GET`: Get a gcp-compute-ssl-certificate by its "name"
- `LIST`: List all gcp-compute-ssl-certificate
- ~~`SEARCH`~~
