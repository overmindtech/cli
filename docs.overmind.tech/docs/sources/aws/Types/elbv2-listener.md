---
title: ELB Listener
sidebar_label: elbv2-listener
---

An Elastic Load Balancing (ELB) v2 Listener is the component of an Application Load Balancer (ALB) or Network Load Balancer (NLB) that checks for connection requests, using a specified protocol and port, and then routes those requests to one or more target groups according to its rules. Each listener belongs to a single load balancer, can have one default action and multiple conditional rules, and is the entry point for traffic into your load-balancing configuration.  
Further details can be found in the AWS documentation: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-listeners.html

**Terrafrom Mappings:**

- `aws_alb_listener.arn`
- `aws_lb_listener.arn`

## Supported Methods

- `GET`: Get an ELB listener by ARN
- ~~`LIST`~~
- `SEARCH`: Search for ELB listeners by load balancer ARN

## Possible Links

### [`elbv2-load-balancer`](/sources/aws/Types/elbv2-load-balancer)

The listener is directly attached to exactly one load balancer. Overmind uses this link to show which ALB or NLB will be affected if the listener configuration is changed or deleted.

### [`elbv2-rule`](/sources/aws/Types/elbv2-rule)

A listener owns a set of rules that determine how incoming requests are evaluated and forwarded. This link exposes those rules so you can trace the impact of modifying conditions, priorities, or actions.

### [`http`](/sources/stdlib/Types/http)

If the listener uses the HTTP or HTTPS protocol, Overmind represents the public-facing endpoint as an `http` item. This allows cross-checking of listener ports with accessible URLs and aids in identifying unintended exposure.

### [`elbv2-target-group`](/sources/aws/Types/elbv2-target-group)

Listener actions forward traffic to one or more target groups. Overmind links these dependencies so you can see which instances, containers, or IPs will receive traffic, helping you assess downstream blast radius.
