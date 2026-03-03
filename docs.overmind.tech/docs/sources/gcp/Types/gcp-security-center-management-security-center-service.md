---
title: GCP Security Center Management Security Center Service
sidebar_label: gcp-security-center-management-security-center-service
---

The **Security Center Service** resource represents the configuration of Security Command Center (SCC) for a particular Google Cloud location.  
Each instance of this resource indicates that SCC is running in the specified region and records the service‐wide settings that govern how findings are ingested, stored and surfaced.  
Official documentation: https://cloud.google.com/security-command-center/docs/reference/security-center-management/rest/v1/projects.locations.securityCenterServices/list

## Supported Methods

- `GET`: Get a gcp-security-center-management-security-center-service by its "locations|securityCenterServices"
- ~~`LIST`~~
- `SEARCH`: Search Security Center services in a location. Use the format "location".

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

A Security Center Service exists **inside** a specific Google Cloud project – the project determines billing, IAM policies and the scope of resources that SCC monitors. The Overmind link lets you pivot from the project to every Security Center Service it has enabled (and vice-versa), helping you see which projects have security monitoring active in each region.
