# Overmind AWS Source Setup

Terraform module that configures an AWS account for
[Overmind](https://overmind.tech) infrastructure discovery. A single
`terraform apply` creates:

1. An IAM role with a read-only policy in the target AWS account
2. A trust policy allowing Overmind to assume the role via STS external ID
3. An Overmind source registration pointing at the role

## Usage

```hcl
provider "overmind" {}

provider "aws" {
  region = "us-east-1"
}

module "overmind_aws_source" {
  source = "overmindtech/aws-source-setup/overmind"

  name = "production"
}
```

## Inputs

| Name | Description | Type | Default | Required |
| --- | --- | --- | --- | --- |
| `name` | Descriptive name for the source in Overmind | `string` | n/a | yes |
| `regions` | AWS regions to discover (defaults to all non-opt-in regions) | `list(string)` | All 17 standard regions | no |
| `role_name` | Name for the IAM role created in this account | `string` | `"overmind-read-only"` | no |
| `tags` | Additional tags to apply to IAM resources | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
| --- | --- |
| `role_arn` | ARN of the created IAM role |
| `source_id` | UUID of the Overmind source |
| `external_id` | AWS STS external ID used in the trust policy |

## Multi-account example

Use AWS provider aliases to onboard several accounts at once:

```hcl
provider "overmind" {}

provider "aws" {
  alias  = "production"
  region = "us-east-1"
  assume_role { role_arn = "arn:aws:iam::111111111111:role/terraform" }
}

provider "aws" {
  alias  = "staging"
  region = "eu-west-1"
  assume_role { role_arn = "arn:aws:iam::222222222222:role/terraform" }
}

module "overmind_production" {
  source = "overmindtech/aws-source-setup/overmind"
  name   = "production"

  providers = {
    aws      = aws.production
    overmind = overmind
  }
}

module "overmind_staging" {
  source  = "overmindtech/aws-source-setup/overmind"
  name    = "staging"
  regions = ["eu-west-1"]

  providers = {
    aws      = aws.staging
    overmind = overmind
  }
}
```

## Authentication

The Overmind provider reads `OVERMIND_API_KEY` from the environment. The API key
must have `sources:write` scope.

The AWS provider must have permissions to create IAM roles and policies in the
target account.

## Requirements

| Name | Version |
| --- | --- |
| terraform | >= 1.5.0 |
| aws | >= 6.0 |
| overmind | >= 0.1.0 |
