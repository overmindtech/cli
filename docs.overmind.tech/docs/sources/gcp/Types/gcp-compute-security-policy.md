---
title: GCP Compute Security Policy
sidebar_label: gcp-compute-security-policy
---

A GCP Compute Security Policy represents a Cloud Armor security policy. It contains an ordered set of layer-7 filtering rules that allow, deny, or rate-limit traffic directed at a load balancer or backend service. By attaching a security policy you can enforce web-application-firewall (WAF) protections, mitigate DDoS attacks, and define custom match conditions—all without changing your application code. Overmind ingests these resources so you can understand how proposed changes will affect the exposure and resilience of your workloads before you deploy them.

For full details see the official Google Cloud documentation: https://cloud.google.com/armor/docs/security-policy-concepts

**Terrafrom Mappings:**

- `google_compute_security_policy.name`

## Supported Methods

- `GET`: Get GCP Compute Security Policy by "gcp-compute-security-policy-name"
- `LIST`: List all GCP Compute Security Policy items
- ~~`SEARCH`~~
