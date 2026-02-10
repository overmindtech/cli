---
title: SSM Parameter
sidebar_label: ssm-parameter
---

AWS Systems Manager (SSM) Parameters, stored in the Systems Manager Parameter Store, provide a centralised, version-controlled repository for configuration data such as plain strings, SecureStrings (encrypted secrets), and hierarchical documents. They allow you to decouple configuration and secrets from code, share settings across services and accounts, and take advantage of fine-grained IAM access controls. See the official AWS documentation for full details: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html

**Terrafrom Mappings:**

- `aws_ssm_parameter.name`
- `aws_ssm_parameter.arn`

## Supported Methods

- `GET`: Get an SSM parameter by name
- `LIST`: List all SSM parameters
- `SEARCH`: Search for SSM parameters by ARN. This supports ARNs from IAM policies that contain wildcards

## Possible Links

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

If a parameter’s value represents an IP address or a list of addresses (for example, a whitelist used by a Lambda function or security group rule generator), Overmind will surface a link to the corresponding `ip` entity so that you can trace where the address originates and what else depends on it.

### [`http`](/sources/stdlib/Types/http)

Parameters often store URLs for upstream APIs, S3 buckets, or internal services. When the value of a parameter matches an HTTP or HTTPS URL, Overmind creates an `http` link, enabling you to follow the dependency chain from the configuration to the external or internal endpoint.

### [`dns`](/sources/stdlib/Types/dns)

Likewise, when a parameter’s value contains a hostname or FQDN, Overmind links it to the relevant `dns` record. This makes it easy to assess the impact of DNS changes on applications that retrieve their endpoint addresses from Parameter Store.
