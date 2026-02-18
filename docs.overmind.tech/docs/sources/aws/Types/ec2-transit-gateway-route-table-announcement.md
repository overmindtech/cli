---
title: Transit Gateway Route Table Announcement
sidebar_label: ec2-transit-gateway-route-table-announcement
---

A Transit Gateway Route Table Announcement represents the advertisement of a transit gateway route table to a peer—for example, to another transit gateway (peering) or to an AWS Network Manager core network. Routes that originate from such an announcement appear in the route table with a `TransitGatewayRouteTableAnnouncementId`, and Overmind links those [ec2-transit-gateway-route](/sources/aws/Types/ec2-transit-gateway-route) items to this type.

Official API documentation: [DescribeTransitGatewayRouteTableAnnouncements](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTableAnnouncements.html)

## Note

Overmind does not currently provide a dedicated adapter for `ec2-transit-gateway-route-table-announcement`. This type is documented because [ec2-transit-gateway-route](/sources/aws/Types/ec2-transit-gateway-route) items can link to it when a route originates from a route table announcement (`TransitGatewayRouteTableAnnouncementId`).
