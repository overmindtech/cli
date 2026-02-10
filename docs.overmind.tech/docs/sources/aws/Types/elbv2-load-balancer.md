---
title: Elastic Load Balancer
sidebar_label: elbv2-load-balancer
---

Elastic Load Balancers distribute incoming traffic across multiple targets, improving the availability and scalability of applications. The “v2” API covers Application, Network and Gateway Load Balancers, each of which can automatically scale to meet demand and provide a single DNS endpoint for users. Full service behaviour and limits are documented in the AWS Elastic Load Balancing User Guide (https://docs.aws.amazon.com/elasticloadbalancing/latest/userguide/).

**Terrafrom Mappings:**

- `aws_lb.arn`
- `aws_lb.id`

## Supported Methods

- `GET`: Get an ELB by name
- `LIST`: List all ELBs
- `SEARCH`: Search for ELBs by ARN

## Possible Links

### [`elbv2-target-group`](/sources/aws/Types/elbv2-target-group)

The load balancer forwards requests to one or more target groups; each listener rule references a target group that contains the actual EC2 instances, IPs or Lambda functions receiving traffic.

### [`elbv2-listener`](/sources/aws/Types/elbv2-listener)

Listeners define the port and protocol that the load balancer accepts and contain the rules that map traffic to target groups; every load balancer has at least one listener.

### [`dns`](/sources/stdlib/Types/dns)

ELBs are accessed via a DNS name (e.g., `my-alb-123456.eu-west-1.elb.amazonaws.com`). External DNS records resolve to the IPs managed by AWS behind this name.

### [`route53-hosted-zone`](/sources/aws/Types/route53-hosted-zone)

Route 53 alias or CNAME records are commonly created in a hosted zone to point a friendly domain name to the load balancer’s DNS name.

### [`ec2-vpc`](/sources/aws/Types/ec2-vpc)

An ELB is deployed inside a specific VPC, inheriting its network boundaries and able to route traffic only within that VPC (except for internet-facing endpoints).

### [`ec2-subnet`](/sources/aws/Types/ec2-subnet)

The load balancer is placed into one or more subnets; for high availability at least two subnets (usually across AZs) are required.

### [`ec2-address`](/sources/aws/Types/ec2-address)

Network Load Balancers can be allocated static Elastic IP addresses, one per subnet, providing fixed public IPs for the load balancer.

### [`ip`](/sources/aws/Types/networkmanager-network-resource-relationship)

Each Elastic IP (for NLB) or the dynamically allocated addresses (for ALB/Gateway LB) represent the underlying IP resources that the DNS name resolves to.

### [`ec2-security-group`](/sources/aws/Types/ec2-security-group)

Application and Gateway Load Balancers are associated with security groups which control the allowed inbound and outbound traffic to the load balancer endpoints.
