---
title: GCP Orgpolicy Policy
sidebar_label: gcp-orgpolicy-policy
---

An Organisation Policy in Google Cloud Platform (GCP) lets administrators enforce or relax specific constraints on GCP resources across the organisation, folder, or project hierarchy. Each policy represents the chosen configuration for a single constraint (for example, restricting service account key creation or limiting the set of permitted VM regions) on a single resource node. By querying an Org Policy, Overmind can reveal whether pending changes will violate existing security or governance rules before deployment.  
Official documentation: https://cloud.google.com/resource-manager/docs/organization-policy/org-policy-constraints

**Terrafrom Mappings:**

- `google_org_policy_policy.name`

## Supported Methods

- `GET`: Get a gcp-orgpolicy-policy by its "name"
- `LIST`: List all gcp-orgpolicy-policy
- `SEARCH`: Search with the full policy name: projects/[project]/policies/[constraint] (used for terraform mapping).
