---
title: ELB Rule
sidebar_label: elbv2-rule
---

An ELBv2 listener rule specifies how an Application Load Balancer (ALB) or Network Load Balancer (NLB) should handle requests that arrive on a particular listener. Each rule has a priority, a set of conditions (for example, host-based or path-based matches) and a set of actions (such as forwarding to a target group, redirecting, or returning a fixed response). When traffic reaches the listener, the load balancer evaluates its rules in priority order and executes the actions associated with the first rule whose conditions are met. Refer to the official AWS documentation for further information: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-listeners.html#listener-rules

**Terrafrom Mappings:**

- `aws_alb_listener_rule.arn`
- `aws_lb_listener_rule.arn`

## Supported Methods

- `GET`: Get a rule by ARN
- ~~`LIST`~~
- `SEARCH`: Search for rules by listener ARN
