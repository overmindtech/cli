---
title: DynamoDB Backup
sidebar_label: dynamodb-backup
---

A DynamoDB Backup represents a point-in-time, fully-managed snapshot of an Amazon DynamoDB table, including all of its data and global secondary indexes. Back-ups can be created on demand or retained automatically through continuous point-in-time recovery (PITR). They allow you to restore the table to any state within the retention window, or to clone the data into a new table in the same or another region for testing and disaster-recovery purposes. For further details, see the official AWS documentation: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/BackupRestore.html

## Supported Methods

- ~~`GET`~~
- `LIST`: List all DynamoDB backups
- `SEARCH`: Search for a DynamoDB backup by table name

## Possible Links

### [`dynamodb-table`](/sources/aws/Types/dynamodb-table)

Each backup is intrinsically tied to the table from which it was taken; Overmind therefore links a `dynamodb-backup` item to its source `dynamodb-table` so you can trace data-protection coverage, understand restore scopes, and assess the blast radius of table changes or deletions.
