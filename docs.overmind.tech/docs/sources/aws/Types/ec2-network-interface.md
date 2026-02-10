---
title: EC2 Network Interface
sidebar_label: ec2-network-interface
---

An Amazon Elastic Compute Cloud (EC2) Network Interface – often referred to as an Elastic Network Interface (ENI) – is a virtual network card that can be attached to an EC2 instance. It provides the instance with connectivity within a Virtual Private Cloud (VPC) and, optionally, to the public Internet. Each ENI contains a primary private IPv4 address, one or more secondary IPv4 addresses, IPv6 addresses if enabled, one or more security groups, a MAC address, and, when required, an Elastic IP address or a public DNS name. ENIs can be moved between instances, created in advance, or used for high-availability network configurations such as dual-homed instances.  
For complete details see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html

**Terrafrom Mappings:**

- `aws_network_interface.id`

## Supported Methods

- `GET`: Get a network interface by ID
- `LIST`: List all network interfaces
- `SEARCH`: Search network interfaces by ARN

## Possible Links

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

An ENI can be attached to an EC2 instance, providing that instance with network connectivity. Overmind links the interface to the instance(s) it is or has been attached to.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Each ENI is associated with one or more security groups. These groups define the inbound and outbound traffic rules applied at the interface level. The link shows which security groups control traffic for the ENI.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

The ENI owns one or more IP addresses (private IPv4, secondary IPv4, IPv6, and optionally Elastic IP). This relationship exposes the individual IP resources attached to the interface.

### [`dns`](/sources/stdlib/Types/dns)

If an ENI has a public IPv4 address, AWS automatically creates a corresponding public DNS name; private DNS names may also be present within the VPC. Overmind links these DNS records to the ENI.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

An ENI is created inside a specific subnet. The subnet determines the address range from which the ENI’s private IPs are allocated and the availability zone in which it resides.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Every ENI exists within a single VPC, inheriting that VPC’s routing tables, DHCP options, and network ACLs. This link shows the parent VPC for the interface.
