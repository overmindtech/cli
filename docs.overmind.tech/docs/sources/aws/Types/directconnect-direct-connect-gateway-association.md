---
title: Direct Connect Gateway Association
sidebar_label: directconnect-direct-connect-gateway-association
---

A Direct Connect Gateway Association represents the attachment of a virtual private gateway (VGW) or a transit gateway (TGW) to an AWS Direct Connect gateway. Once associated, the on-premises network that is connected through an AWS Direct Connect dedicated or hosted connection can reach the VPCs behind the VGW/TGW, even if they are in different AWS Regions.  
For more detail, see the AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/direct-connect-gateways-intro.html#direct-connect-gateway-associations

**Terraform Mappings:**

- `aws_dx_gateway_association.id`

## Supported Methods

- `GET`: Get a direct connect gateway association by direct connect gateway ID and virtual gateway ID
- ~~`LIST`~~
- `SEARCH`: Search direct connect gateway associations by direct connect gateway ID

## Possible Links

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

A Direct Connect Gateway Association is a child resource of a Direct Connect Gateway, so every association is linked to the Direct Connect Gateway to which the VGW/TGW is attached.
