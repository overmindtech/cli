---
title: GCP Compute Ssl Policy
sidebar_label: gcp-compute-ssl-policy
---

A Google Cloud Compute **SSL Policy** specifies the minimum TLS protocol version and the set of supported cipher suites that HTTPS or SSL-proxy load balancers are allowed to use when negotiating SSL/TLS with clients. By attaching an SSL Policy to a target HTTPS proxy or target SSL proxy, you can enforce stronger security standards, ensure compliance, and gradually deprecate outdated encryption algorithms without disrupting traffic.  
For detailed information, refer to the official Google Cloud documentation: https://cloud.google.com/load-balancing/docs/ssl-policies-concepts.

**Terrafrom Mappings:**

* `google_compute_ssl_policy.name`

## Supported Methods

* `GET`: Get a gcp-compute-ssl-policy by its "name"
* `LIST`: List all gcp-compute-ssl-policy
* ~~`SEARCH`~~
