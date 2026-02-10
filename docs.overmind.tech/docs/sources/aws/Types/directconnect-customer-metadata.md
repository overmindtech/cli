---
title: Customer Metadata
sidebar_label: directconnect-customer-metadata
---

Customer Metadata represents the customer agreement that is on file for your AWS account in relation to AWS Direct Connect. The record contains information such as the name and Amazon Resource Name (ARN) of the agreement, its current revision and status, and the Region in which the agreement applies. Being able to inspect this resource lets you confirm that the correct contractual terms have been accepted before you attempt to create or modify Direct Connect connections, helping you avoid deployment failures that stem from missing or outdated agreements.  
For further details see the AWS API documentation: https://docs.aws.amazon.com/directconnect/latest/APIReference/API_DescribeCustomerMetadata.html

## Supported Methods

- `GET`: Get a customer agreement by name
- `LIST`: List all customer agreements
- `SEARCH`: Search customer agreements by ARN
