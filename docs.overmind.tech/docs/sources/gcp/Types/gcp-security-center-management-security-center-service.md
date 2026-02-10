---
title: GCP Security Center Management Security Center Service
sidebar_label: gcp-security-center-management-security-center-service
---

A Security Center Service resource represents the activation and configuration of Google Cloud Security Command Center (SCC) for a particular location (for example `europe-west2`) within a project, folder, or organisation. It records whether SCC is enabled, the current service tier (Standard, Premium, or Enterprise), and other operational metadata such as activation time and billing status. Administrators use this resource to programme-matically enable or disable SCC, upgrade or downgrade the service tier, and verify the health of the service across all regions.  
Official documentation: https://cloud.google.com/security-command-center/docs/reference/security-center-management/rest/v1/folders.locations.securityCenterServices#SecurityCenterService

## Supported Methods

- `GET`: Get a gcp-security-center-management-security-center-service by its "locations|securityCenterServices"
- ~~`LIST`~~
- `SEARCH`: Search Security Center services in a location. Use the format "location".
