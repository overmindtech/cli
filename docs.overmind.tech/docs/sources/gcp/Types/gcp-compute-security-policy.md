---
title: GCP Compute Security Policy
sidebar_label: gcp-compute-security-policy
---

A GCP Compute Security Policy represents a Google Cloud Armor security policy that you configure to protect your applications and services from malicious or unwanted traffic. Each policy is made up of an ordered list of rules that allow, deny, or rate-limit requests based on layer-3/4 characteristics or custom layer-7 expressions. Security policies can be associated with external Application Load Balancers, Cloud CDN, and other HTTP(S)-based backend services, enabling centralised, declarative control over inbound traffic behaviour.  
For full details, see the official Google documentation: https://cloud.google.com/compute/docs/reference/rest/v1/securityPolicies

**Terrafrom Mappings:**

* `google_compute_security_policy.name`

## Supported Methods

* `GET`: Get GCP Compute Security Policy by "gcp-compute-security-policy-name"
* `LIST`: List all GCP Compute Security Policy items
* ~~`SEARCH`~~
