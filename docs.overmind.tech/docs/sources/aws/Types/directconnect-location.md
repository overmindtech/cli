---
title: Direct Connect Location
sidebar_label: directconnect-location
---

An AWS Direct Connect Location represents one of the globally distributed, carrier-neutral data-centre facilities where you can order and terminate an AWS Direct Connect dedicated circuit. Each location has a unique location code that you reference when requesting a connection, viewing available port speeds, generating LOAs, or validating the physical site of an existing circuit. Understanding which locations are available – and the risks or constraints linked to each – helps you design resilient, low-latency connectivity between your on-premises network and AWS.  
For full details see the official AWS documentation: https://docs.aws.amazon.com/directconnect/latest/UserGuide/WorkingWithLocations.html

**Terrafrom Mappings:**

- `aws_dx_location.location_code`

## Supported Methods

- `GET`: Get a Location by its code
- `LIST`: List all Direct Connect Locations
- `SEARCH`: Search Direct Connect Locations by ARN
