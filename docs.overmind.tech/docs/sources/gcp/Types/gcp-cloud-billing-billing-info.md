---
title: GCP Cloud Billing Billing Info
sidebar_label: gcp-cloud-billing-billing-info
---

`gcp-cloud-billing-billing-info` represents a Google Cloud **ProjectBillingInfo** resource, i.e. the object that records which Cloud Billing Account a particular GCP project is attached to and whether billing is currently enabled.  
Knowing which Billing Account is used – and whether charges can actually accrue – is often vital when assessing the financial risk of a new deployment.  
Official documentation: https://cloud.google.com/billing/docs/reference/rest/v1/projects/getBillingInfo

## Supported Methods

- `GET`: Get a gcp-cloud-billing-billing-info by its "name"
- ~~`LIST`~~
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

Every ProjectBillingInfo belongs to exactly one Cloud project. Overmind therefore links the `gcp-cloud-billing-billing-info` item to the corresponding `gcp-cloud-resource-manager-project` item, allowing you to trace billing-account associations back to the project that will generate the spend.
