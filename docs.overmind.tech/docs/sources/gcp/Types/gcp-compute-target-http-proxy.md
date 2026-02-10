---
title: GCP Compute Target Http Proxy
sidebar_label: gcp-compute-target-http-proxy
---

A Google Cloud Compute Target HTTP Proxy acts as the intermediary between a forwarding rule and your defined URL map. When an incoming request reaches the load balancer, the proxy evaluates the host and path rules in the URL map and then forwards the request to the selected backend service. In essence, it is the control point that translates external client traffic into internal service calls, supporting features such as global anycast IPs, health-checking, and intelligent request routing for high-availability web applications.  
For further information, see the official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpProxies

**Terrafrom Mappings:**

- `google_compute_target_http_proxy.name`

## Supported Methods

- `GET`: Get a gcp-compute-target-http-proxy by its "name"
- `LIST`: List all gcp-compute-target-http-proxy
- ~~`SEARCH`~~
