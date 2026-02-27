---
title: GCP Compute Ssl Certificate
sidebar_label: gcp-compute-ssl-certificate
---

A **Google Compute SSL Certificate** represents an SSL certificate resource that can be attached to Google Cloud load-balancers to provide encrypted (HTTPS or SSL proxy) traffic termination. It stores the public certificate and its corresponding private key, enabling Compute Engine and Cloud Load Balancing to serve traffic securely on the specified domains. Certificates can be self-managed (you upload the PEM-encoded certificate and key) or Google-managed (Google provisions and renews them automatically). Full details are available in the official documentation: [Google Compute Engine – SSL certificates](https://cloud.google.com/compute/docs/reference/rest/v1/sslCertificates).

**Terrafrom Mappings:**

* `google_compute_ssl_certificate.name`

## Supported Methods

* `GET`: Get a gcp-compute-ssl-certificate by its "name"
* `LIST`: List all gcp-compute-ssl-certificate
* ~~`SEARCH`~~
