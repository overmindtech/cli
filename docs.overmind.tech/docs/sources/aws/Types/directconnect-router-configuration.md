---
title: Router Configuration
sidebar_label: directconnect-router-configuration
---

AWS Direct Connect can automatically generate a sample configuration that you can paste into the customer-side router that terminates a private, public or transit virtual interface. The Router Configuration object represents that text file. Because the template is created by AWS specifically for the selected virtual interface it already contains the correct BGP ASN, VLAN, IP addressing and other parameters for the connection, reducing the chance of a mis-configuration.  
Full details of the API can be found in the AWS Direct Connect API Reference: https://docs.aws.amazon.com/directconnect/latest/APIReference/API_DescribeRouterConfiguration.html

**Terrafrom Mappings:**

- `aws_dx_router_configuration.virtual_interface_id`

## Supported Methods

- `GET`: Get a Router Configuration by Virtual Interface ID
- ~~`LIST`~~
- `SEARCH`: Search Router Configuration by ARN

## Possible Links

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

A Router Configuration is generated for, and therefore has a **1-to-1** relationship with, a Direct Connect Virtual Interface. The link allows you to navigate from the virtual interface to the exact configuration you should apply to your on-premises router (and vice-versa), making it easier to validate that the interface has been deployed according to the recommended configuration.
