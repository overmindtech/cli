---
title: API Gateway Domain Name
sidebar_label: apigateway-domain-name
---

An AWS API Gateway Domain Name represents a custom DNS name (e.g. `api.example.com`) that you attach to one or more stages of a REST, HTTP or WebSocket API. By creating this resource you can present a branded, user-friendly endpoint instead of the default `*.execute-api.<region>.amazonaws.com` host, configure an ACM or imported TLS certificate, choose an edge-optimised or regional endpoint, enable mutual TLS and define API mappings. Further information can be found in the official documentation: https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-create-custom-domain-name.html

**Terrafrom Mappings:**

- `aws_api_gateway_domain_name.domain_name`

## Supported Methods

- `GET`: Get a Domain Name by domain-name
- `LIST`: List Domain Names
- `SEARCH`: Search Domain Names by ARN
