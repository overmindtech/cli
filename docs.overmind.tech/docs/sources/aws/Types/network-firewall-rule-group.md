---
title: Network Firewall Rule Group
sidebar_label: network-firewall-rule-group
---

AWS Network Firewall Rule Groups are reusable collections of stateless or stateful inspection rules that you attach to a Network Firewall policy. They let you define, version, and manage traffic-inspection logic independently from the firewalls that enforce it. A rule group may contain Suricata-compatible stateful rules, 5-tuple stateless rules, or a combination of both, and can optionally be encrypted with a customer-managed AWS KMS key. See the official AWS documentation for full details: https://docs.aws.amazon.com/network-firewall/latest/developerguide/rule-groups.html

**Terrafrom Mappings:**

- `aws_networkfirewall_rule_group.name`

## Supported Methods

- `GET`: Get a Network Firewall Rule Group by name
- `LIST`: List Network Firewall Rule Groups
- `SEARCH`: Search for Network Firewall Rule Groups by ARN

## Possible Links

### [`kms-key`](/sources/aws/Types/kms-key)

If the rule group was created with an `EncryptionConfiguration`, the ARN of the customer-managed KMS key used for encryption is stored in the resource metadata. Overmind therefore links the rule group to the corresponding `kms-key` item.

### [`sns-topic`](/sources/aws/Types/sns-topic)

Operational teams often configure CloudWatch alarms on Network Firewall metrics that publish to an SNS topic; the alarm definition contains the rule group ARN as a dimension. When such a relationship exists, Overmind links the rule group to the `sns-topic` so that users can trace alerting pathways.

### [`network-firewall-rule-group`](/sources/aws/Types/network-firewall-rule-group)

Firewall policies can reference multiple rule groups, and a single rule group can be associated with several policies. Overmind records these associations, allowing one rule group to be linked to other rule groups that are attached to the same policy or that replace it through versioning.
