---
title: DynamoDB Table
sidebar_label: dynamodb-table
---

Amazon DynamoDB is AWS’s fully-managed NoSQL database service, providing single-millisecond latency at virtually any scale. A DynamoDB table is the primary container for data, storing items as key–value pairs and supporting features such as on-demand or provisioned capacity, global replication, streams and automatic encryption at rest.  
For a full description of table capabilities, limits and API operations, see the official AWS documentation: https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Table.html

**Terrafrom Mappings:**

- `aws_dynamodb_table.arn`

## Supported Methods

- `GET`: Get a DynamoDB table by name
- `LIST`: List all DynamoDB tables
- `SEARCH`: Search for DynamoDB tables by ARN

## Possible Links

### [`dynamodb-table`](/sources/aws/Types/dynamodb-table)

When a table participates in a global table configuration, each regional replica is represented as a separate `dynamodb-table` item. Overmind links these peer replicas so that you can see the full set of regions involved in the same globally replicated table.

### [`kms-key`](/sources/aws/Types/kms-key)

If server-side encryption is enabled with a customer-managed KMS key, the table is linked to the `kms-key` that protects its data. This allows you to trace encryption dependencies and assess the impact of key rotation or deletion.
