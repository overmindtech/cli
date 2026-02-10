---
title: Classic Load Balancer
sidebar_label: elb-load-balancer
---

A Classic Load Balancer (CLB) is the original generation of AWS Elastic Load Balancing. It automatically distributes incoming application or network traffic across multiple Amazon EC2 instances that are located in one or more Availability Zones, improving fault-tolerance and scalability. A CLB provides either HTTP/HTTPS or TCP load balancing and exposes a single DNS end-point that clients connect to.  
Official documentation: https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/introduction.html

**Terrafrom Mappings:**

- `aws_elb.arn`

## Supported Methods

- `GET`: Get a classic load balancer by name
- `LIST`: List all classic load balancers
- `SEARCH`: Search for classic load balancers by ARN

## Possible Links

### [`dns`](/sources/stdlib/Types/dns)

The load balancer’s endpoint is presented as a DNS A/AAAA/CNAME record (e.g. `my-clb-123456.eu-west-2.elb.amazonaws.com`). Overmind links the CLB to this DNS record so that you can see which hostname is exposed publicly.

### [`route53-hosted-zone`](/sources/aws/Types/route53-hosted-zone)

AWS hosts the CLB DNS name inside an Amazon-owned Route 53 hosted zone, and you may also create alias or CNAME records in your own hosted zones that point to the CLB. The link shows every hosted zone that contains records referencing the load balancer.

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

A Classic Load Balancer must be attached to one or more subnets in each Availability Zone where it is enabled. This link reveals the exact subnets the CLB is deployed into.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

Because the selected subnets belong to a specific VPC, the CLB itself resides inside that VPC. The link allows you to trace the load balancer back to its enclosing network boundary.

### [`ec2-instance`](/sources/aws/Types/ec2-instance)

Backend EC2 instances are registered with the CLB as targets. Overmind lists every registered instance so you can assess what workloads will receive traffic from the load balancer.

### [`elb-instance-health`](/sources/aws/Types/elb-instance-health)

For each registered EC2 instance AWS maintains per-target health information (healthy, unhealthy, etc.). This link surfaces those health objects, letting you understand why particular instances may not be receiving traffic.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

In a VPC, a Classic Load Balancer is associated with one or more security groups that govern allowed inbound and outbound traffic. Overmind links to these security groups so you can inspect the firewall rules that protect the load balancer.
