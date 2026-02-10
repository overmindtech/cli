---
title: GCP Service Usage Service
sidebar_label: gcp-service-usage-service
---

Represents an individual Google Cloud API or service (for example, `pubsub.googleapis.com`, `compute.googleapis.com`) that can be enabled or disabled within a project or folder via the Service Usage API.  
It holds metadata such as the service’s name, state (ENABLED, DISABLED, etc.), configuration and any consumer-specific settings. Managing this resource controls whether dependent resources in the project are allowed to operate.  
Official documentation: https://cloud.google.com/service-usage/docs/overview

## Supported Methods

- `GET`: Get a gcp-service-usage-service by its "name"
- `LIST`: List all gcp-service-usage-service
- ~~`SEARCH`~~

## Possible Links

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

A Pub/Sub topic can only exist and function if the `pubsub.googleapis.com` service is ENABLED in the same project. Overmind links a `gcp-service-usage-service` whose name is `pubsub.googleapis.com` to all `gcp-pub-sub-topic` resources in that project so that you can assess the blast radius of disabling the API.
