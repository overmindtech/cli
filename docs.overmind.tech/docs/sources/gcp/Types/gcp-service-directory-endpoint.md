---
title: GCP Service Directory Endpoint
sidebar_label: gcp-service-directory-endpoint
---

A **Service Directory Endpoint** represents a concrete network endpoint (host/IP address and port) that implements a Service Directory service within a namespace and location. Clients resolve a service and obtain one or more endpoints in order to make network calls. Endpoints can carry arbitrary key-value metadata and may point at instances running inside a VPC, on-premises, or in another cloud.  
Official documentation: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints

**Terrafrom Mappings:**

* `google_service_directory_endpoint.id`

## Supported Methods

* `GET`: Get a gcp-service-directory-endpoint by its "locations|namespaces|services|endpoints"
* ~~`LIST`~~
* `SEARCH`: Search for endpoints by "location|namespace_id|service_id" or "projects/[project_id]/locations/[location]/namespaces/[namespace_id]/services/[service_id]/endpoints/[endpoint_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A Service Directory endpoint’s address usually resides within a VPC network. Linking an endpoint to its `gcp-compute-network` resource lets you trace which network the IP belongs to, ensuring that connectivity policies (firewalls, routes, private service access, etc.) permit clients to reach the service before deployment.
