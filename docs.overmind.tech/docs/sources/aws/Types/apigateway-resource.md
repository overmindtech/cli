---
title: API Gateway
sidebar_label: apigateway-resource
---

An **API Gateway Resource** represents a single path segment within an Amazon API Gateway REST API. Each resource forms part of the hierarchical URL structure of your API and can have HTTP methods (such as GET, POST, DELETE) attached to it, along with integrations, authorisers and request/response models. Correctly mapping these resources is critical because mis-configured paths can expose unintended back-ends or shadow existing routes. Overmind pulls every API Gateway Resource into its graph so you can understand how proposed changes will affect downstream services before you deploy them.  
For further details, refer to the official AWS documentation: https://docs.aws.amazon.com/apigateway/latest/api/API_Resource.html

**Terrafrom Mappings:**

- `aws_api_gateway_resource.id`

## Supported Methods

- `GET`: Get a Resource by rest-api-id/resource-id
- ~~`LIST`~~
- `SEARCH`: Search Resources by REST API ID
