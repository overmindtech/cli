---
title: GCP Service Usage Service
sidebar_label: gcp-service-usage-service
---

A **Service Usage Service** represents an individual Google-managed API or service (e.g. `compute.googleapis.com`, `pubsub.googleapis.com`) and its enablement state inside a single GCP project. By querying this resource you can determine whether a particular service is currently enabled, disabled, or in another transitional state for that project, which is critical for understanding if downstream resources can be created successfully.  
Official documentation: https://cloud.google.com/service-usage/docs/reference/rest/v1/services

## Supported Methods

* `GET`: Get a gcp-service-usage-service by its "name"
* `LIST`: List all gcp-service-usage-service
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

Every Service Usage Service exists **within** a single Cloud Resource Manager project. The project acts as the parent container and dictates billing, IAM policies and quota that apply to the service.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

A Pub/Sub topic can only be created or used if the **`pubsub.googleapis.com`** Service Usage Service is enabled in the same project. Overmind links the topic back to its enabling service so you can quickly spot configuration drift or missing API enablement that would prevent deployment.
