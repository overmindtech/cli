---
title: Virtual Interface
sidebar_label: directconnect-virtual-interface
---

A Virtual Interface (VIF) is the logical layer that sits on top of an AWS Direct Connect physical connection and provides Layer 3 access into AWS. Three flavours are available—private, public and transit—each supporting different routing destinations and services. A VIF defines the VLAN, BGP peering IPs, Autonomous System Numbers (ASNs), jumbo-frame settings and, optionally, a Direct Connect Gateway association.  
Official AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/WorkingWithVirtualInterfaces.html

**Terrafrom Mappings:**

- `aws_dx_private_virtual_interface.id`
- `aws_dx_public_virtual_interface.id`
- `aws_dx_transit_virtual_interface.id`

## Supported Methods

- `GET`: Get a virtual interface by ID
- `LIST`: List all virtual interfaces
- `SEARCH`: Search virtual interfaces by connection ID

## Possible Links

### [`directconnect-connection`](/sources/aws/Types/directconnect-connection)

Every VIF must be created against a Direct Connect physical connection. The link lets you trace which circuit (location, port speed, AWS account) the virtual interface is riding on.

### [`directconnect-direct-connect-gateway`](/sources/aws/Types/directconnect-direct-connect-gateway)

Private and transit VIFs can be attached to a Direct Connect Gateway to reach multiple VPCs or on-premises networks. This link shows that association, helping you see the downstream network blast-radius of a VIF change.

### [`rdap-ip-network`](/sources/stdlib/Types/rdap-ip-network)

The BGP peer IPs configured on a VIF belong to specific IPv4/IPv6 networks. Linking to RDAP IP network objects allows visibility of route-origin information and public registration data for those peer addresses.

### [`directconnect-direct-connect-gateway-attachment`](/sources/aws/Types/directconnect-direct-connect-gateway-attachment)

When a VIF is associated with a Direct Connect Gateway, an attachment resource is created in AWS. This link maps the VIF to its attachment object so you can understand and audit that relationship.

### [`directconnect-virtual-interface`](/sources/aws/Types/directconnect-virtual-interface)

Some organisations create multiple VIFs on the same physical connection for isolation (e.g., production vs. test). Overmind links sibling VIFs so you can view parallel logical circuits that share the same underlay.
