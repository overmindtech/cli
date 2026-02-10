---
title: IAM Instance Profile Association
sidebar_label: ec2-iam-instance-profile-association
---

An IAM Instance Profile Association represents the live binding between an Amazon EC2 instance and an IAM instance profile (which in turn wraps an IAM role). The association determines which IAM permissions the instance receives via its metadata service. Only one profile can be associated with an instance at a time; changing the association effectively swaps the role that the instance assumes.  
For further information see the AWS API reference: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_IamInstanceProfileAssociation.html

## Supported Methods

- `GET`: Get an IAM Instance Profile Association by ID
- `LIST`: List all IAM Instance Profile Associations
- `SEARCH`: Search IAM Instance Profile Associations by ARN

## Possible Links

### [`iam-instance-profile`](/sources/aws/Types/iam-instance-profile)

The association points to exactly one IAM instance profile, identifying the set of IAM permissions that will be handed to the EC2 instance.

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

Each association belongs to a single EC2 instance, indicating which profile (and hence which role) the instance is currently using.
