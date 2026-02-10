---
title: GCP Compute Ssl Policy
sidebar_label: gcp-compute-ssl-policy
---

Google Cloud SSL policies allow you to define which TLS protocol versions and cipher suites can be used when clients negotiate secure connections with Google Cloud load balancers. By attaching an SSL policy to an HTTPS, SSL, or TCP proxy load balancer, you can enforce modern cryptographic standards, disable deprecated protocols, or maintain compatibility with legacy clients, thereby controlling the security posture of your services. Overmind can surface potential risks—such as the continued availability of weak ciphers—before you deploy.  
For more information, see the official Google Cloud documentation: [SSL policies overview](https://cloud.google.com/compute/docs/reference/rest/v1/sslPolicies/get).

**Terrafrom Mappings:**

- `google_compute_ssl_policy.name`

## Supported Methods

- `GET`: Get a gcp-compute-ssl-policy by its "name"
- `LIST`: List all gcp-compute-ssl-policy
- ~~`SEARCH`~~
