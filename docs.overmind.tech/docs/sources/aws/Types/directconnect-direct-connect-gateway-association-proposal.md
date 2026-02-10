---
title: Direct Connect Gateway Association Proposal
sidebar_label: directconnect-direct-connect-gateway-association-proposal
---

An AWS Direct Connect Gateway Association Proposal represents a cross-account request to attach a Virtual Private Gateway (VGW) or Transit Gateway (TGW) to an existing Direct Connect Gateway (DXGW).  
The proposal is created by the owner of the VGW/TGW and must be accepted by the DXGW owner before the association is established. It contains details such as allowed prefixes and the identifiers of the gateways involved, providing both parties with a clear record of what will change once the proposal is accepted.  
For more information, see the official AWS API documentation: https://docs.aws.amazon.com/directconnect/latest/APIReference/API_CreateDirectConnectGatewayAssociationProposal.html

**Terrafrom Mappings:**

- `aws_dx_gateway_association_proposal.id`

## Supported Methods

- `GET`: Get a Direct Connect Gateway Association Proposal by ID
- `LIST`: List all Direct Connect Gateway Association Proposals
- `SEARCH`: Search Direct Connect Gateway Association Proposals by ARN

## Possible Links

### [`directconnect-direct-connect-gateway-association`](/sources/aws/Types/directconnect-direct-connect-gateway-association)

A proposal, once accepted, becomes a Direct Connect Gateway Association. Therefore, every accepted `directconnect-direct-connect-gateway-association-proposal` will have a corresponding `directconnect-direct-connect-gateway-association` resource that represents the live attachment between the DXGW and the VGW/TGW.
