---
title: CloudFront Key Group
sidebar_label: cloudfront-key-group
---

A CloudFront Key Group is an Amazon CloudFront configuration object that aggregates several public keys under a single identifier. CloudFront uses the keys in the group to verify the signatures on signed URLs, signed cookies, or JSON Web Tokens that you employ to control access to private content. By attaching a key group to a distribution or cache behaviour you can centrally manage which public keys are trusted; adding or removing a key from the group immediately changes who can generate valid signatures without the need to touch individual distributions.  
For more information, refer to the AWS documentation on Key Groups: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/PrivateContent.html#PrivateContent-KeyGroups

**Terrafrom Mappings:**

- `aws_cloudfront_key_group.id`

## Supported Methods

- `GET`: Get a CloudFront Key Group by ID
- `LIST`: List CloudFront Key Groups
- `SEARCH`: Search CloudFront Key Groups by ARN
