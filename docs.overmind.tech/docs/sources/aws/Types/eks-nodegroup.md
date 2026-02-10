---
title: EKS Nodegroup
sidebar_label: eks-nodegroup
---

Amazon EKS managed node groups are a higher-level abstraction that simplifies the provision and lifecycle management of the worker nodes that run your Kubernetes pods. Instead of creating and operating the underlying Amazon EC2 instances yourself, you declare the desired configuration (instance types, scaling parameters, AMI, etc.) and EKS creates and manages an Auto Scaling group on your behalf. See the official AWS documentation for full details: https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html

**Terrafrom Mappings:**

- `aws_eks_node_group.id`

## Supported Methods

- `GET`: Get a node group by unique name (`{clusterName}:{NodegroupName}`)
- ~~`LIST`~~
- `SEARCH`: Search for node groups by cluster name

## Possible Links

### [`ec2-key-pair`](/sources/aws/Types/ec2-key-pair)

If “remote access” is enabled, a node group references an EC2 key pair to allow SSH access to the worker nodes. This creates a dependency on the specified key pair.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Each node group attaches one or more security groups to the network interfaces of its nodes. These security groups control inbound and outbound traffic to the worker nodes.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

When you create a node group you must provide a list of subnets where the nodes will be launched. The node group therefore depends on, and is constrained by, the networking configuration of those subnets.

### [`autoscaling-auto-scaling-group`](/sources/aws/Types/autoscaling-auto-scaling-group)

Behind the scenes, a managed node group is realised as an Auto Scaling group. Changes to the node group propagate directly to its underlying Auto Scaling group.

### [`ec2-launch-template`](/sources/aws/Types/ec2-launch-template)

You can optionally supply a custom launch template to define advanced EC2 settings (user data, tags, block-device mappings, etc.) for the nodes. When used, the node group links to that launch template.
