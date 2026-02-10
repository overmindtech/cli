---
title: Cluster Role Binding
sidebar_label: ClusterRoleBinding
---

A ClusterRoleBinding grants the permissions defined in a `ClusterRole` to one or more subjects (users, groups, or ServiceAccounts) across the entire Kubernetes cluster. Whereas a `RoleBinding` is namespace-scoped, a ClusterRoleBinding has cluster-wide effect, making it a critical component of RBAC configuration.
For further details, see the Kubernetes documentation: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding

**Terrafrom Mappings:**

- `kubernetes_cluster_role_binding_v1.metadata[0].name`
- `kubernetes_cluster_role_binding.metadata[0].name`

## Supported Methods

- `GET`: Get a Cluster Role Binding by name
- `LIST`: List all Cluster Role Bindings
- `SEARCH`: Search for a Cluster Role Binding using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`ClusterRole`](/sources/k8s/Types/ClusterRole)

The ClusterRoleBinding’s `roleRef` field points to the name of a `ClusterRole`. Overmind represents this relationship so you can trace which set of permissions (rules) is being granted cluster-wide.

### [`ServiceAccount`](/sources/k8s/Types/ServiceAccount)

If a ClusterRoleBinding contains one or more ServiceAccounts in its `subjects` array, Overmind links the binding to those ServiceAccounts, allowing you to see exactly which workload identities receive the referenced cluster-level permissions.
