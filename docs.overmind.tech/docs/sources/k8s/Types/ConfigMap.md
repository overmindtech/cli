---
title: Config Map
sidebar_label: ConfigMap
---

A ConfigMap is a Kubernetes API object used to store non-confidential configuration data in key-value pairs. It allows you to decouple environment-specific configuration from your container images so that the same image can be reused in different environments with different settings. Pods and other Kubernetes workloads can consume the data held in a ConfigMap as environment variables, command-line arguments or configuration files mounted into a volume. For an in-depth overview, see the official documentation: https://kubernetes.io/docs/concepts/configuration/configmap/

**Terrafrom Mappings:**

- `kubernetes_config_map_v1.metadata[0].name`
- `kubernetes_config_map.metadata[0].name`

## Supported Methods

- `GET`: Get a Config Map by name
- `LIST`: List all Config Maps
- `SEARCH`: Search for a Config Map using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
