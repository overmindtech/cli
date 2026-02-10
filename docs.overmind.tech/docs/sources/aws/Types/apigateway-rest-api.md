---
title: REST API
sidebar_label: apigateway-rest-api
---

AWS API Gateway REST APIs allow you to build, deploy and manage REST-style interfaces that front your application logic, Lambda functions or other AWS services. A REST API in API Gateway represents the top-level container for all stages, resources, methods, authorisers and deployments that make up your service. Once created, the API can be exposed publicly or kept private behind a VPC endpoint, throttled, monitored and versioned across stages.  
For full details, refer to the official AWS documentation: https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html

**Terrafrom Mappings:**

- `aws_api_gateway_rest_api.id`

## Supported Methods

- `GET`: Get a REST API by ID
- `LIST`: List all REST APIs
- `SEARCH`: Search for REST APIs by their name

## Possible Links

### [`ec2-vpc-endpoint`](/sources/aws/Types/ec2-vpc-endpoint)

If the REST API is configured as a private API, it is exposed inside a VPC through an Interface VPC Endpoint. Overmind links the `apigateway-rest-api` resource to the corresponding `ec2-vpc-endpoint` to show which endpoint clients inside the VPC must use to reach the API and to surface any network-level risks (such as missing security-group rules).

### [`apigateway-resource`](/sources/aws/Types/apigateway-resource)

An API Gateway REST API is composed of one or more resources, each representing a path segment (for example `/users` or `/orders/{orderId}`). Overmind links the parent `apigateway-rest-api` to each individual `apigateway-resource` so you can trace how a request traverses the API hierarchy and identify unprotected or redundant paths.
