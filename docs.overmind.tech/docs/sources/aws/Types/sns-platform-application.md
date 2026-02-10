---
title: SNS Platform Application
sidebar_label: sns-platform-application
---

An Amazon Simple Notification Service (SNS) **platform application** represents a collection of credentials that allow SNS to send push notifications through a specific mobile push service, such as Apple APNS, Google FCM or Amazon ADM. Once you create a platform application, you can register individual mobile devices (platform endpoints) under it and publish messages that will be delivered to those devices by the relevant push provider.  
For a full description see the AWS documentation: https://docs.aws.amazon.com/sns/latest/dg/mobile-push-send.html#mobile-push-sns-platform.

**Terrafrom Mappings:**

- `aws_sns_platform_application.id`

## Supported Methods

- `GET`: Get an SNS platform application by its ARN
- `LIST`: List all SNS platform applications
- `SEARCH`: Search SNS platform applications by ARN

## Possible Links

### [`sns-endpoint`](/sources/aws/Types/sns-endpoint)

Each platform application can have many child **SNS platform endpoints**—one per registered device. Linking the application to its endpoints lets Overmind surface which devices are affected by configuration changes or credential mis-configurations in the parent application.
