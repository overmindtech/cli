---
title: EKS Cluster
sidebar_label: eks-cluster
---

Amazon Elastic Kubernetes Service (EKS) is a managed Kubernetes control plane that allows you to run Kubernetes workloads on AWS without the operational overhead of managing the underlying master nodes. An EKS cluster handles tasks such as control-plane provisioning, scalability, high availability and automatic patching, while letting you attach one or more node groups (either managed or self-managed) to run your containerised applications. See the official AWS documentation for full details: https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html

**Terraform Mappings:**

- `aws_eks_cluster.arn`

## Supported Methods

- `GET`: Get a cluster by name
- `LIST`: List all clusters
- `SEARCH`: Search for clusters by ARN
