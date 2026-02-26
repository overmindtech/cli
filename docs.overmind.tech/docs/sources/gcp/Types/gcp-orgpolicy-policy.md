---
title: GCP Orgpolicy Policy
sidebar_label: gcp-orgpolicy-policy
---

An Organisation Policy (orgpolicy) in Google Cloud is a resource that applies a constraint to part of the resource hierarchy (organisation, folder, or project). It allows administrators to enforce governance rules—such as restricting the regions in which resources may be created, blocking the use of certain services, or mandating specific network configurations—before workloads are deployed.  
For full details see Google’s official documentation: https://cloud.google.com/resource-manager/docs/organization-policy/overview

**Terrafrom Mappings:**

  * `google_org_policy_policy.name`

## Supported Methods

* `GET`: Get a gcp-orgpolicy-policy by its "name"
* `LIST`: List all gcp-orgpolicy-policy
* `SEARCH`: Search with the full policy name: projects/[project]/policies/[constraint] (used for terraform mapping).

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

A project is one of the resource hierarchy levels to which an Organisation Policy can be attached. Each gcp-orgpolicy-policy documented here is therefore linked to the gcp-cloud-resource-manager-project that the policy governs.