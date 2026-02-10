---
title: Lambda Layer
sidebar_label: lambda-layer
---

AWS Lambda Layers are a packaging construct used to share code, data, and runtimes between multiple Lambda functions. A layer is published once and can then be referenced by any function in the same AWS account (or, if shared, by functions in other accounts), keeping deployment packages small and ensuring that common dependencies are managed in a single place. Overmind surfaces Lambda Layers so that you can see which functions depend on them and understand the blast radius of any proposed change.  
For full details, see the official AWS documentation: https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html

## Supported Methods

- ~~`GET`~~
- `LIST`: List all lambda layers
- ~~`SEARCH`~~

## Possible Links

### [`lambda-layer-version`](/sources/aws/Types/lambda-layer-version)

A Lambda Layer can have multiple immutable versions; this link shows the individual versions that belong to the parent layer.
