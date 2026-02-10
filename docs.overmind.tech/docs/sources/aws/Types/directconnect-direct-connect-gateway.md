---
title: Direct Connect Gateway
sidebar_label: directconnect-direct-connect-gateway
---

An AWS Direct Connect gateway is a global virtual routing resource that allows you to attach one or more Direct Connect private virtual interfaces to one or more Virtual Private Gateways (VGWs) or Transit Gateways (TGWs) across any AWS Region (with the exception of the AWS China Regions). By decoupling the physical Direct Connect connection from a specific VPC or Region, it simplifies multi-region and multi-account network architectures, provides centralised route control, and reduces the number of BGP sessions that need to be managed.  
For a detailed overview, refer to the official AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/direct-connect-gateways.html

**Terrafrom Mappings:**

- `aws_dx_gateway.id`

## Supported Methods

- `GET`: Get a direct connect gateway by ID
- `LIST`: List all direct connect gateways
- `SEARCH`: Search direct connect gateway by ARN
