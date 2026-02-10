---
title: Key Pair
sidebar_label: ec2-key-pair
---

An Amazon EC2 Key Pair is a set of cryptographic keys that enables secure, password-less SSH access to your EC2 instances and other compatible services. The public key is stored in AWS, while the private key is downloaded and managed by you. If the private key is compromised or lost, access to the associated instances is at risk, so tracking key pairs is critical for security posture.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html

**Terrafrom Mappings:**

- `aws_key_pair.id`

## Supported Methods

- `GET`: Get a key pair by name
- `LIST`: List all key pairs
- `SEARCH`: Search for key pairs by ARN
