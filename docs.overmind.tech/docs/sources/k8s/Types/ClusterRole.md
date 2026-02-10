---
title: Cluster Role
sidebar_label: ClusterRole
---

A ClusterRole is a non-namespaced Kubernetes RBAC resource that groups together one or more policy rules, defining which verbs (such as `get`, `list`, `create`, `delete`) are allowed on which resources across the entire cluster. Because it is cluster-scoped, the permissions it grants apply to all namespaces. It can be referenced by a `RoleBinding` (to limit its scope to a single namespace) or by a `ClusterRoleBinding` (to apply it cluster-wide) and is commonly used to grant system-level or cross-namespace permissions to users, service accounts or other principals.
For full details, see the official Kubernetes documentation: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#clusterrole

**Terrafrom Mappings:**

- `kubernetes_cluster_role_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a Cluster Role by name
- `LIST`: List all Cluster Roles
- `SEARCH`: Search for a Cluster Role using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
