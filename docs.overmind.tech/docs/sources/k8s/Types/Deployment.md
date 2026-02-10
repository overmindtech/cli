---
title: Deployment
sidebar_label: Deployment
---

A Deployment is a higher-level Kubernetes workload resource that declaratively manages a set of identical Pods by creating and maintaining the appropriate number of ReplicaSets. With a Deployment you describe the desired state—such as how many replicas should be running or which Pod template to use—and the Kubernetes control plane continually works to bring the actual state in line with that specification. Deployments support rolling updates, rollbacks, and pausing/resuming of updates, making them the most common mechanism for managing stateless applications on Kubernetes clusters.  
For the complete specification see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/

**Terrafrom Mappings:**

- `kubernetes_deployment_v1.metadata[0].name`
- `kubernetes_deployment.metadata[0].name`

## Supported Methods

- `GET`: Get a Deployment by name
- `LIST`: List all Deployments
- `SEARCH`: Search for a Deployment using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`ReplicaSet`](/sources/k8s/Types/ReplicaSet)

Each Deployment automatically creates and owns one or more ReplicaSets. The ReplicaSet is responsible for keeping the specified number of Pod replicas running, while the Deployment supervises the ReplicaSets, deciding when to create new ones or scale them to facilitate updates or rollbacks.
