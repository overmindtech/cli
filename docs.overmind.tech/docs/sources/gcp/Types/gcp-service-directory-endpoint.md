---
title: GCP Service Directory Endpoint
sidebar_label: gcp-service-directory-endpoint
---

A Service Directory Endpoint represents a concrete network destination that backs a Service Directory Service inside Google Cloud. Each endpoint records the IP address, port and (optionally) metadata that client workloads use to discover and call the service. Endpoints are created inside a hierarchy of **Project → Location → Namespace → Service → Endpoint** and are resolved at run-time through Service Directory’s DNS or HTTP APIs, allowing producers to register instances and consumers to discover them without hard-coding addresses.  
Official documentation: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints

**Terrafrom Mappings:**

- `google_service_directory_endpoint.id`

## Supported Methods

- `GET`: Get a gcp-service-directory-endpoint by its "locations|namespaces|services|endpoints"
- ~~`LIST`~~
- `SEARCH`: Search for endpoints by "location|namespace_id|service_id" or "projects/[project_id]/locations/[location]/namespaces/[namespace_id]/services/[service_id]/endpoints/[endpoint_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Each endpoint is associated with a specific VPC network; the `network` field determines from which network the endpoint can be reached and which clients can resolve it. When Overmind discovers a Service Directory Endpoint, it links the item to the corresponding gcp-compute-network so you can trace service discovery issues back to network configuration or segmentation problems.
