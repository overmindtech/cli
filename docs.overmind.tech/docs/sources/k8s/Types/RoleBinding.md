---
title: Role Binding
sidebar_label: RoleBinding
---

A Kubernetes **RoleBinding** grants the permissions defined in a Role (or ClusterRole) to a set of subjects—users, groups or service accounts—within a single namespace. It is a cornerstone object in Kubernetes RBAC, controlling who can perform which actions on namespaced resources. See the official Kubernetes documentation for full details: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding

**Terrafrom Mappings:**

- `kubernetes_role_binding.metadata[0].name`
- `kubernetes_role_binding_v1.metadata[0].name`

## Supported Methods

- `GET`: Get a RoleBinding by name
- `LIST`: List all RoleBindings
- `SEARCH`: Search for a RoleBinding using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`Role`](/sources/k8s/Types/Role)

The RoleBinding points to a Role via the `roleRef` field. This link lets Overmind trace which set of rules (verbs, resources, API groups) will be granted when the RoleBinding is applied.

### [`ClusterRole`](/sources/k8s/Types/ClusterRole)

Although scoped to a namespace, a RoleBinding can reference a ClusterRole instead of a Role. Overmind links the two so you can see when cluster-wide permission sets are being delegated into a namespace.

### [`ServiceAccount`](/sources/k8s/Types/ServiceAccount)

Service accounts commonly appear in the `subjects` list of a RoleBinding. Linking these enables Overmind to reveal which workloads (pods using the service account) will inherit the referenced permissions.
