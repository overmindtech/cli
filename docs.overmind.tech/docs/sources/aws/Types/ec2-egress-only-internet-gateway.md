---
title: Egress Only Internet Gateway
sidebar_label: ec2-egress-only-internet-gateway
---

An Egress Only Internet Gateway (EOIGW) is a horizontally-scaled, highly available AWS VPC component that allows outbound-only IPv6 traffic from your VPC to the internet while preventing unsolicited inbound connections. Unlike a standard Internet Gateway, an EOIGW supports IPv6 traffic exclusively and enforces one-way egress, making it a useful control when you want resources such as application servers to reach external IPv6 services without being directly reachable from the internet.  
For detailed information, see the official AWS documentation: https://docs.aws.amazon.com/vpc/latest/userguide/egress-only-internet-gateway.html

**Terrafrom Mappings:**

- `egress_only_internet_gateway.id`

## Supported Methods

- `GET`: Get an egress only internet gateway by ID
- `LIST`: List all egress only internet gateways
- `SEARCH`: Search egress only internet gateways by ARN

## Possible Links

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

An EOIGW is attached to exactly one VPC. Overmind represents this relationship so that you can navigate from a VPC to its associated egress-only internet gateways and understand which networks can initiate outbound IPv6 traffic to the internet.
