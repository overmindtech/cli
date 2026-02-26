---
title: GCP Compute Target Http Proxy
sidebar_label: gcp-compute-target-http-proxy
---

A **GCP Compute Target HTTP Proxy** routes incoming HTTP requests to the appropriate backend service based on rules defined in a URL map. It terminates the client connection, consults the associated `google_compute_url_map`, and then forwards traffic to the selected backend (for example, a backend service or serverless NEG). Target HTTP proxies are a key component of Google Cloud external HTTP(S) Load Balancing.  
See the official documentation for full details: https://cloud.google.com/load-balancing/docs/target-proxies#target_http_proxy

**Terrafrom Mappings:**

  * `google_compute_target_http_proxy.name`

## Supported Methods

* `GET`: Get a gcp-compute-target-http-proxy by its "name"
* `LIST`: List all gcp-compute-target-http-proxy
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-url-map`](/sources/gcp/Types/gcp-compute-url-map)

A Target HTTP Proxy must reference exactly one URL map. Overmind uses this link to trace from the proxy to the URL map that defines its routing rules, enabling you to understand and surface any risks associated with misconfigured path matchers or backend services.