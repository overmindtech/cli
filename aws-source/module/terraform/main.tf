data "overmind_aws_external_id" "this" {}

resource "aws_iam_role" "overmind" {
  name = var.role_name

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::942836531449:root" }
        Action    = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = data.overmind_aws_external_id.this.external_id
          }
        }
      },
      {
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::942836531449:root" }
        Action    = "sts:TagSession"
      },
    ]
  })

  tags = merge(var.tags, {
    "overmind.version" = "2026-02-17"
  })
}

resource "aws_iam_role_policy" "overmind" {
  name = "OvmReadOnly"
  role = aws_iam_role.overmind.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "apigateway:Get*",
          "autoscaling:Describe*",
          "cloudfront:Get*",
          "cloudfront:List*",
          "cloudwatch:Describe*",
          "cloudwatch:GetMetricData",
          "cloudwatch:ListTagsForResource",
          "directconnect:Describe*",
          "dynamodb:Describe*",
          "dynamodb:List*",
          "ec2:Describe*",
          "ecs:Describe*",
          "ecs:List*",
          "eks:Describe*",
          "eks:List*",
          "elasticfilesystem:Describe*",
          "elasticloadbalancing:Describe*",
          "iam:Get*",
          "iam:List*",
          "kms:Describe*",
          "kms:Get*",
          "kms:List*",
          "lambda:Get*",
          "lambda:List*",
          "network-firewall:Describe*",
          "network-firewall:List*",
          "networkmanager:Describe*",
          "networkmanager:Get*",
          "networkmanager:List*",
          "rds:Describe*",
          "rds:ListTagsForResource",
          "route53:Get*",
          "route53:List*",
          "s3:GetBucket*",
          "s3:ListAllMyBuckets",
          "sns:Get*",
          "sns:List*",
          "sqs:Get*",
          "sqs:List*",
          "ssm:Describe*",
          "ssm:Get*",
          "ssm:ListTagsForResource",
        ]
        Resource = "*"
      },
    ]
  })
}

resource "overmind_aws_source" "this" {
  name         = var.name
  aws_role_arn = aws_iam_role.overmind.arn
  aws_regions  = var.regions
}
