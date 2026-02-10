---
title: Fargate Profile
sidebar_label: eks-fargate-profile
---

An Amazon EKS Fargate profile tells EKS which pods in a cluster should run on AWS Fargate rather than on self-managed or managed EC2 worker nodes. It contains a set of selectors (namespace and optional labels) and the networking configuration (subnets and the pod execution IAM role) that EKS will use when it launches Fargate tasks on your behalf. See the official documentation for full details: https://docs.aws.amazon.com/eks/latest/userguide/fargate-profile.html

**Terrafrom Mappings:**

- `aws_eks_fargate_profile.id`

## Supported Methods

- `GET`: Get a fargate profile by unique name (`{clusterName}:{FargateProfileName}`)
- ~~`LIST`~~
- `SEARCH`: Search for fargate profiles by cluster name

## Possible Links

### [`iam-role`](/sources/aws/Types/iam-role)

Each Fargate profile references a “pod execution role”, an IAM role that grants EKS permission to pull container images and publish pod logs when it provisions the Fargate tasks. Overmind therefore creates a link from the profile to the IAM role specified in `pod_execution_role_arn`.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

The profile’s `subnet_ids` field defines the VPC subnets into which the Fargate pods will be launched. Overmind links the profile to every subnet listed, helping you trace network reachability and security-group inheritance for the pods that will run under this profile.
