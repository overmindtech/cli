---
title: Launch Template Version
sidebar_label: ec2-launch-template-version
---

An AWS EC2 Launch Template Version is an immutable snapshot of all the parameters that make up a particular revision of an EC2 launch template – such as AMI ID, instance type, network interfaces, storage, tags and user-data. Each version can be referenced directly when launching instances or by services like Auto Scaling, Spot Fleets and EC2 Fleet, giving you reproducible, auditable instance configuration.
For full details see the AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_LaunchTemplateVersion.html

## Supported Methods

- `GET`: Get a launch template version by `{templateId}.{version}`
- `LIST`: List all launch template versions
- `SEARCH`: Search launch template versions by ARN

## Possible Links

### [`ec2-network-interface`](/sources/aws/Types/ec2-network-interface)

The version can embed zero or more network interface specifications, each of which becomes an `ec2-network-interface` when an instance is launched from the template.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

Within the network interface or placement settings the version may reference a specific subnet ID, tying the launched instance to that `ec2-subnet`.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Security group IDs listed in the template control inbound and outbound traffic for instances started from this version, linking it to the relevant `ec2-security-group` resources.

### [`ec2-image`](/sources/aws/Types/ec2-image)

Every launch template version specifies an AMI ID, creating a dependency on the corresponding `ec2-image`.

### [`ec2-key-pair`](/sources/aws/Types/ec2-key-pair)

If a key name is supplied, the version references an `ec2-key-pair` used for SSH access to Linux instances or password encryption for Windows instances.

### [`ec2-snapshot`](/sources/aws/Types/ec2-snapshot)

EBS block-device mappings in the template can point to snapshot IDs, establishing a relationship with the relevant `ec2-snapshot` objects.

### [`ec2-capacity-reservation`](/sources/aws/Types/ec2-capacity-reservation)

The template may include a capacity reservation target, associating the version with a specific `ec2-capacity-reservation`.

### [`ec2-placement-group`](/sources/aws/Types/ec2-placement-group)

Placement settings in the version can name a placement group, indicating that instances should launch into the linked `ec2-placement-group`.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Static private or public IP addresses specified in the network interface configuration will be materialised as `ip` resources when the template version is used to launch an instance.
