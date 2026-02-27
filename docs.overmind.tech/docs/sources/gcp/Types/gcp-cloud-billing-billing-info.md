---
title: GCP Cloud Billing Billing Info
sidebar_label: gcp-cloud-billing-billing-info
---

The **Cloud Billing – Billing Info** resource represents the billing configuration that is attached to an individual Google Cloud project.  
For a given project it records which Cloud Billing Account is linked, whether billing is currently enabled, and other metadata that controls how usage costs are charged.  
The resource is surfaced by the Cloud Billing API endpoint  
`cloudbilling.googleapis.com/v1/projects/{projectId}/billingInfo`.  
Full details are available in the official Google documentation:  
https://cloud.google.com/billing/docs/reference/rest/v1/projects/getBillingInfo

Knowing the contents of this object allows Overmind to determine, for example, whether a project is running with an unexpectedly disabled billing account or whether it is tied to the correct cost centre before a deployment is made.

## Supported Methods

* `GET`: Get a gcp-cloud-billing-billing-info by its "name"
* ~~`LIST`~~
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

Every Billing Info object belongs to exactly one Cloud Resource Manager Project.  
Overmind creates a link from `gcp-cloud-billing-billing-info` → `gcp-cloud-resource-manager-project` so that users can trace the billing configuration back to the workload and other resources that live inside the same project.
