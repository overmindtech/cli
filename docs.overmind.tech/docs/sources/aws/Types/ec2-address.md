---
title: EC2 Address
sidebar_label: ec2-address
---

An EC2 Address represents an Elastic IP (EIP) in AWS – a static, public IPv4 address that you can allocate to your account and assign to running resources such as EC2 instances or network interfaces. Elastic IPs let you mask the failure of a single instance by rapidly remapping the address to another resource, ensuring minimal disruption to services that rely on a fixed public endpoint. See the official AWS documentation for full details: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html

**Terrafrom Mappings:**

- `aws_eip.public_ip`
- `aws_eip_association.public_ip`

## Supported Methods

- `GET`: Get an EC2 address by Public IP
- `LIST`: List EC2 addresses
- `SEARCH`: Search for EC2 addresses by ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

An Elastic IP can be attached directly to an EC2 instance; this link shows which instance currently holds (or most recently held) the address, allowing you to trace external reachability back to the compute resource.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

The Elastic IP is ultimately a routable IPv4 address; this link connects the high-level EIP object to the underlying IP entity so that you can track dependencies and overlap with other networking resources in your estate.

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

When an Elastic IP is associated with an EC2 instance, it is actually bound to one of the instance’s network interfaces (ENIs). This link identifies the specific ENI, enabling deeper analysis of traffic flow, security groups, and subnet placement that pertain to the EIP.
