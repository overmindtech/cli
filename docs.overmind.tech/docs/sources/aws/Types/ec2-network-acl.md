---
title: Network ACL
sidebar_label: ec2-network-acl
---

A Network Access Control List (ACL) is a stateless, virtual firewall that controls inbound and outbound traffic at the subnet boundary within an Amazon Virtual Private Cloud (VPC). Each rule in a Network ACL is evaluated in order, enabling or denying traffic based on protocol, port range and source or destination IP. Unlike security groups, Network ACLs apply to all resources inside the associated subnets, making them a coarse-grained layer of network security.  
For full details, see the AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-network-acls.html

**Terrafrom Mappings:**

- `aws_network_acl.id`

## Supported Methods

- `GET`: Get a network ACL
- `LIST`: List all network ACLs
- `SEARCH`: Search for network ACLs by ARN

## Possible Links

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

A Network ACL is attached to one or more subnets; traffic entering or leaving those subnets is evaluated against the ACL’s rule set. Overmind therefore links an `ec2-network-acl` to the `ec2-subnet` resources it governs.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Every Network ACL exists inside a single VPC. Overmind links an `ec2-network-acl` to its parent `ec2-vpc` to show the broader network context in which the ACL operates.
