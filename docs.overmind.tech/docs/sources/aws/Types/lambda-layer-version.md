---
title: Lambda Layer Version
sidebar_label: lambda-layer-version
---

AWS Lambda Layer Version represents an immutable, version-numbered snapshot of a Lambda layer—an archive of shared code, libraries, custom runtimes or other assets that can be attached to multiple Lambda functions. Each time you publish a layer you create a new layer version, referenced in the form `arn:aws:lambda:<region>:<account-id>:layer:<layer-name>:<version-number>`. Using layers helps decouple shared dependencies from individual function packages, streamline updates and encourage code reuse across your serverless estate.
Further details can be found in the official AWS documentation: https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html

**Terrafrom Mappings:**

- `aws_lambda_layer_version.arn`

## Supported Methods

- `GET`: Get a layer version by full name (`{layerName}:{versionNumber}`)
- ~~`LIST`~~
- `SEARCH`: Search for layer versions by ARN
