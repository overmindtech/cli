---
title: Service Account
sidebar_label: ServiceAccount
---

A ServiceAccount is a Kubernetes resource that provides an identity to processes running inside Pods, allowing them to authenticate to the Kubernetes API and other services with the minimum privileges required. Each ServiceAccount can be linked to one or more Secrets that store its bearer token or image-pull credentials, and these Secrets are automatically mounted into Pods that specify the ServiceAccount. Further information can be found in the official Kubernetes documentation: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/.

**Terrafrom Mappings:**

- `kubernetes_service_account.metadata[0].name`
- `kubernetes_service_account_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a ServiceAccount by name
- `LIST`: List all ServiceAccounts
- `SEARCH`: Search for a ServiceAccount using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Secret`](/sources/k8s/Types/Secret)

A ServiceAccount is associated with Secrets that hold its authentication token or are referenced in `imagePullSecrets`. These Secrets determine how Pods using the ServiceAccount authenticate to the cluster or to external registries, making them critical for understanding access scopes and potential risk.
