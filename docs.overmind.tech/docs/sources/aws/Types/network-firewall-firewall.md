---
title: Network Firewall
sidebar_label: network-firewall-firewall
---

AWS Network Firewall is a managed, stateful, layer-4 and layer-7 firewall service that you deploy inside your own Amazon Virtual Private Cloud (VPC). It lets you inspect and filter both inbound and outbound traffic by applying rule groups that you author or obtain from third-party providers. Because the service is fully managed, AWS handles availability, scaling and patching, allowing you to focus on writing network-security rules rather than on the underlying infrastructure. For a full overview, see the official documentation: https://docs.aws.amazon.com/network-firewall/latest/developerguide/what-is-aws-network-firewall.html

**Terrafrom Mappings:**

- `aws_networkfirewall_firewall.name`

## Supported Methods

- `GET`: Get a Network Firewall by name
- `LIST`: List Network Firewalls
- `SEARCH`: Search for Network Firewalls by ARN

## Possible Links

### [`network-firewall-firewall-policy`](/sources/aws/Types/network-firewall-firewall-policy)

Each Network Firewall is associated with exactly one firewall policy, which defines the stateful and stateless rule groups, default actions and logging configuration that the firewall must enforce.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

A firewall is deployed into one or more dedicated subnets—known as firewall subnets—within the VPC. These subnets host the firewall endpoints that inspect traffic traversing the Availability Zones.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

The firewall operates inside a specific VPC, inspecting traffic that enters, leaves or moves within that VPC according to the routing configuration you set up.

### [`s3-bucket`](/sources/aws/Types/s3-bucket)

You can configure Network Firewall to export alert and flow logs to an Amazon S3 bucket for long-term storage, auditing or further analysis; the bucket therefore becomes a downstream logging destination for the firewall.

### [`iam-policy`](/sources/aws/Types/iam-policy)

Creation, modification and deletion of Network Firewall resources are controlled through IAM policies. These policies grant or deny the required `network-firewall:*` permissions to principals such as users, roles and service accounts.

### [`kms-key`](/sources/aws/Types/kms-key)

If you choose to encrypt log data that Network Firewall delivers to Amazon S3 or CloudWatch Logs with a customer-managed key, the firewall references an AWS KMS key. The key is used for server-side encryption of the exported log objects.
