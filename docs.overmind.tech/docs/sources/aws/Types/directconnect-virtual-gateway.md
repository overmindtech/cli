---
title: Direct Connect Virtual Gateway
sidebar_label: directconnect-virtual-gateway
---

A Direct Connect virtual gateway (sometimes called a virtual private gateway, or **VGW**) is the AWS-managed end-point that terminates a private virtual interface and presents it to your Amazon VPC. It provides the control-plane for routing traffic between your on-premises network and one or more VPCs over an AWS Direct Connect link, removing the need to run VPN hardware or BGP sessions inside the VPC itself. By querying this resource, Overmind can show you which VPCs and Direct Connect virtual interfaces are affected, surface any missing or insecure route advertisements, and highlight configuration drift _before_ changes are deployed.

For more information, refer to the AWS Direct Connect documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/virtual-gateway.html

## Supported Methods

- `GET`: Get a virtual gateway by ID
- `LIST`: List all virtual gateways
- `SEARCH`: Search virtual gateways by ARN
