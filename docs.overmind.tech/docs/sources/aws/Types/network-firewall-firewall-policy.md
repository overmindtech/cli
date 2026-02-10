---
title: Network Firewall Policy
sidebar_label: network-firewall-firewall-policy
---

An AWS Network Firewall Policy is the central configuration object that tells the AWS Network Firewall service how to inspect, filter, and log traffic that flows through a firewall. The policy groups together references to stateless and stateful rule groups, sets default actions for traffic that does not match a rule, and can optionally attach TLS inspection configurations. Multiple firewalls can share the same policy, making it easy to apply a consistent security posture across different VPCs or accounts.  
For full service documentation, see the official AWS docs: https://docs.aws.amazon.com/network-firewall/latest/developerguide/firewall-policies.html

**Terrafrom Mappings:**

- `aws_networkfirewall_firewall_policy.name`

## Supported Methods

- `GET`: Get a Network Firewall Policy by name
- `LIST`: List Network Firewall Policies
- `SEARCH`: Search for Network Firewall Policies by ARN

## Possible Links

### [`network-firewall-rule-group`](/sources/aws/Types/network-firewall-rule-group)

A firewall policy is essentially a collection of references to stateless and stateful rule groups. Each rule group defined under the policy dictates how specific traffic patterns are handled. Overmind links a policy to its rule groups so that you can quickly understand which inspection rules are being applied.

### [`network-firewall-tls-inspection-configuration`](/sources/aws/Types/network-firewall-tls-inspection-configuration)

If the policy includes a TLS inspection configuration, encrypted traffic can be decrypted, inspected, and then re-encrypted. Overmind links the policy to any associated TLS inspection configurations to show whether the firewall is capable of deep packet inspection for TLS flows.

### [`kms-key`](/sources/aws/Types/kms-key)

Firewall policies may specify a KMS key for the encryption of log data or stateful rule group data at rest. Overmind surfaces this link so that you can assess the cryptographic controls protecting your firewall’s sensitive data.
