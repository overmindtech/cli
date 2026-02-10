---
title: SNS Endpoint
sidebar_label: sns-endpoint
---

The SNS Endpoint resource represents a single destination—typically a mobile device, browser, or desktop application instance—that can receive push notifications through Amazon Simple Notification Service (SNS). Each endpoint is created under a specific Platform Application and is identified by a unique Amazon Resource Name (ARN). Managing endpoints correctly is crucial, as inactive or mis-configured endpoints can lead to failed deliveries, increased costs, or even unwanted data exposure. For full details see the official AWS documentation: https://docs.aws.amazon.com/sns/latest/dg/mobile-push-send-devicetoken.html

## Supported Methods

- `GET`: Get an SNS endpoint by its ARN
- ~~`LIST`~~
- `SEARCH`: Search SNS endpoints by associated Platform Application ARN
