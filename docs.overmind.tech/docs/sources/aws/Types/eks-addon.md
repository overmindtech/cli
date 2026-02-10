---
title: EKS Addon
sidebar_label: eks-addon
---

An Amazon EKS Addon is an AWS-managed installation of common operational software—such as CoreDNS, kube-proxy, the Amazon VPC CNI plugin or the Amazon EBS CSI driver—onto an Amazon Elastic Kubernetes Service (EKS) cluster. Addons let you declare the component, version and configuration you want, while AWS takes care of deployment, upgrades, security patches and ongoing lifecycle management. Using addons keeps the cluster’s critical services consistent and up to date without manual intervention.
For more information, see the official AWS documentation on EKS Add-ons: https://docs.aws.amazon.com/eks/latest/userguide/eks-add-ons.html

**Terrafrom Mappings:**

- `aws_eks_addon.id`

## Supported Methods

- `GET`: Get an addon by unique name (`{clusterName}:{addonName}`)
- ~~`LIST`~~
- `SEARCH`: Search addons by cluster name
