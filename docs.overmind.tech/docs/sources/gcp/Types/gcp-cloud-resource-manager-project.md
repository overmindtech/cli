---
title: GCP Cloud Resource Manager Project
sidebar_label: gcp-cloud-resource-manager-project
---

A Google Cloud Resource Manager Project represents the fundamental organisational unit within Google Cloud Platform (GCP). Every compute, storage or networking asset you create must live inside a Project, which in turn sits under a Folder or Organisation node. Projects provide isolated boundaries for Identity and Access Management (IAM), quotas, billing, API enablement and lifecycle operations such as creation, update, suspension and deletion. By modelling Projects, Overmind can surface risks linked to mis-scoped IAM roles, neglected billing settings or interactions with other resources *before* any change is pushed to production.  
Official documentation: https://cloud.google.com/resource-manager/docs/creating-managing-projects

## Supported Methods

* `GET`: Get a gcp-cloud-resource-manager-project by its "name"
* ~~`LIST`~~
* ~~`SEARCH`~~
