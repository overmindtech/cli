---
title: Node
sidebar_label: Node
---

A Kubernetes **Node** is a worker machine (virtual or physical) that runs the Pods making up a cluster’s workloads. Each Node contains the services necessary to run containers, including the container runtime, kubelet and kube-proxy, and is managed by the Kubernetes control plane. For more details see the official Kubernetes documentation: https://kubernetes.io/docs/concepts/architecture/nodes/

**Terrafrom Mappings:**

- `kubernetes_node_taint.metadata[0].name`

## Supported Methods

- `GET`: Get a Node by name
- `LIST`: List all Nodes
- `SEARCH`: Search for a Node using the ListOptions JSON format e.g. `{"labelSelector": "app=wordpress"}`

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

A Node is discoverable in the cluster via its DNS entry. Overmind links the Node to its corresponding DNS record(s) so you can trace how applications or services resolve to this worker machine.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Every Node advertises one or more internal and external IP addresses. Overmind establishes a link between the Node resource and these IP objects to surface network reachability or exposure risks.

### [`ec2-volume`](/sources/aws/Types/ec2-volume)

When Kubernetes is running on AWS, Nodes (EC2 instances) may have EBS volumes attached to provide persistent storage for Pods. Overmind links the Node to the `ec2-volume` resources it mounts, allowing you to evaluate storage-related blast radius or compliance concerns.
