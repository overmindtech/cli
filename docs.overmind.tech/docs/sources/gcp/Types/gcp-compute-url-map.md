---
title: GCP Compute Url Map
sidebar_label: gcp-compute-url-map
---

A Google Cloud Platform (GCP) Compute URL Map is the routing table used by an External or Internal HTTP(S) Load Balancer. It evaluates the host and path of each incoming request and, according to the host rules and path matchers you configure, forwards that request to the appropriate backend service or backend bucket. In other words, the URL map determines “which traffic goes where” once it reaches the load balancer, making it a critical part of any web-facing deployment.  
Official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/urlMaps

**Terrafrom Mappings:**

- `google_compute_url_map.name`

## Supported Methods

- `GET`: Get a gcp-compute-url-map by its "name"
- `LIST`: List all gcp-compute-url-map
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

Each URL map references one or more backend services in its path-matcher rules. Overmind therefore creates outbound links from a `gcp-compute-url-map` to every `gcp-compute-backend-service` that might receive traffic, allowing you to trace the full request path and identify downstream risks.
