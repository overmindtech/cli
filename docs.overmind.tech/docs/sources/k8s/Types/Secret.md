---
title: Secret
sidebar_label: Secret
---

A Kubernetes Secret is an object that holds a small amount of sensitive data—such as passwords, tokens, or keys—so that it can be used by Pods without being written to image or configuration files. Storing confidential information in a Secret allows you to keep it separate from application code and to control how and when it is exposed to the running workload. For a detailed overview, see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/configuration/secret/

**Terrafrom Mappings:**

- `kubernetes_secret_v1.metadata[0].name`
- `kubernetes_secret.metadata[0].name`

## Supported Methods

- `GET`: Get a Secret by name
- `LIST`: List all Secrets
- `SEARCH`: Search for a Secret using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
