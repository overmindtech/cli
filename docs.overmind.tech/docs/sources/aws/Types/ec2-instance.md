---
title: EC2 Instance
sidebar_label: ec2-instance
---

An Amazon EC2 instance is a resizable virtual server that runs in the AWS cloud and provides the compute layer of most workloads. Instances can be started, stopped, terminated, resized and placed into different networking or storage configurations, allowing you to run applications without purchasing physical hardware. For full details see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Instances.html

**Terrafrom Mappings:**

- `aws_instance.id`
- `aws_instance.arn`

## Supported Methods

- `GET`: Get an EC2 instance by ID
- `LIST`: List all EC2 instances
- `SEARCH`: Search EC2 instances by ARN

## Possible Links

### [`ec2-instance-status`](/sources/aws/Types/ec2-instance-status)

Represents the current state of the instance (pending, running, stopping, stopped, etc.), health checks, and scheduled events.

### [`iam-instance-profile`](/sources/aws/Types/iam-instance-profile)

An instance can be launched with an IAM instance profile, enabling the software running on it to assume a role and gain AWS permissions.

### [`ec2-capacity-reservation`](/sources/aws/Types/ec2-capacity-reservation)

If the instance is launched into a specific capacity reservation, that reservation object is linked here to show the source of reserved compute capacity.

### [`ec2-image`](/sources/aws/Types/ec2-image)

Every EC2 instance is created from an Amazon Machine Image (AMI). This link points to the AMI used at launch time.

### [`ec2-key-pair`](/sources/aws/Types/ec2-key-pair)

For Linux and some Windows instances a key pair is specified for SSH/RDP access; the referenced key pair is linked here.

### [`ec2-placement-group`](/sources/aws/Types/ec2-placement-group)

Instances can be placed in a placement group to influence network performance or availability. This link shows that relationship.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Each instance receives one or more private and, optionally, public IP addresses. These addresses are surfaced as separate `ip` resources linked to the instance.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

The instance’s primary network interface is attached to a specific subnet; that subnet is linked to reveal networking context.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

The subnet (and thus the instance) resides inside a VPC. Linking the VPC shows the broader network boundary and associated routing.

### [`dns`](/sources/stdlib/Types/dns)

Public and private DNS names resolve to the instance’s IP addresses; these DNS records are connected through this link.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

One or more security groups control inbound and outbound traffic to the instance network interfaces. Those groups are linked here.

### [`ec2-volume`](/sources/aws/Types/ec2-volume)

EBS volumes attached to the instance for root and additional block storage are represented and linked by this type.
