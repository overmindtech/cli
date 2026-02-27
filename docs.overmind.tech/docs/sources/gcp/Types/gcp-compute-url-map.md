---
title: GCP Compute Url Map
sidebar_label: gcp-compute-url-map
---

A Google Cloud Platform (GCP) Compute URL Map is a routing table used by HTTP(S) load balancers to decide where an incoming request should be sent. It matches the request’s host name and URL path to a set of rules and then forwards the traffic to the appropriate backend service or backend bucket. URL Maps make it possible to implement advanced traffic-management patterns such as domain-based and path-based routing, default fall-back targets, and traffic migration between versions of a service.  
Official documentation: https://cloud.google.com/load-balancing/docs/url-map-concepts

**Terrafrom Mappings:**

* `google_compute_url_map.name`

## Supported Methods

* `GET`: Get a gcp-compute-url-map by its "name"
* `LIST`: List all gcp-compute-url-map
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-backend-service`](/sources/gcp/Types/gcp-compute-backend-service)

A URL Map points to one or more backend services as its routing targets. Each rule in the map specifies which `gcp-compute-backend-service` should receive the traffic that matches the rule’s host and path conditions.
