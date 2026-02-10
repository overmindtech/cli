---
title: Role
sidebar_label: Role
---

A Kubernetes Role is an RBAC (Role-Based Access Control) resource that defines a set of permissions, expressed as rules, that apply within a single namespace. By binding a Role to a Subject (user, group, or service account) you control which verbs (get, list, create, delete, etc.) can be performed on which API resources inside that namespace. Roles are therefore central to enforcing the principle of least privilege in cluster security.  
See the official Kubernetes documentation for full details: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole

**Terrafrom Mappings:**

- `kubernetes_role_v1.metadata[0].name`
- `kubernetes_role.metadata[0].name`

## Supported Methods

- `GET`: Get a Role by name
- `LIST`: List all Roles
- `SEARCH`: Search for a Role using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`
