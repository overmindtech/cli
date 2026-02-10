---
title: Resource Quota
sidebar_label: ResourceQuota
---

A Kubernetes **ResourceQuota** object allows cluster administrators to limit the aggregate consumption of compute resources (such as CPU and memory), storage, and object counts (Pods, Services, PersistentVolumeClaims, etc.) within a namespace. By defining upper bounds, a ResourceQuota helps prevent any single team or workload from exhausting shared cluster capacity, and encourages fair usage across tenants. When a namespace has one or more quotas in place, resources are checked at creation or update time; if the requested amount would exceed the quota the operation is rejected.  
Official documentation: https://kubernetes.io/docs/concepts/policy/resource-quotas/

**Terrafrom Mappings:**

- `kubernetes_resource_quota_v1.metadata[0].name`
- `kubernetes_resource_quota.metadata[0].name`

## Supported Methods

- `GET`: Get a Resource Quota by name
- `LIST`: List all Resource Quotas
- `SEARCH`: Search for a Resource Quota using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
