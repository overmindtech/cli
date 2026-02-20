---
title: Configure with Terraform
sidebar_position: 2
---

The [Overmind Terraform module](https://registry.terraform.io/modules/overmindtech/aws-source/overmind) configures an AWS account for Overmind infrastructure discovery in a single `terraform apply`. It creates an IAM role with a read-only policy, sets up the trust relationship, and registers the source with Overmind's API. The module is fully compatible with [OpenTofu](https://opentofu.org/).

## Prerequisites

- **Overmind API key** with `sources:write` scope. Create one in [Settings > API Keys](https://app.overmind.tech/settings/api-keys).
- **AWS credentials** with permission to create IAM roles and policies in the target account.
- **Terraform >= 1.5.0** or **OpenTofu >= 1.6.0**.

## Quick Start

```hcl
provider "overmind" {}

provider "aws" {
  region = "us-east-1"
}

module "overmind_aws_source" {
  source = "overmindtech/aws-source/overmind"

  name = "production"
}

output "role_arn" {
  value = module.overmind_aws_source.role_arn
}

output "source_id" {
  value = module.overmind_aws_source.source_id
}
```

Then run:

```bash
export OVERMIND_API_KEY="your-api-key"
terraform init
terraform plan
terraform apply
```

## Authentication

### Overmind Provider

The Overmind provider reads `OVERMIND_API_KEY` from the environment. The API key must have `sources:write` scope.

You can also set it in the provider block:

```hcl
provider "overmind" {
  api_key = var.overmind_api_key
}
```

### AWS Provider

The AWS provider must have permissions to create IAM roles and policies in the target account. Any standard AWS authentication method works (environment variables, shared credentials file, SSO, etc.). See the [AWS provider documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration) for details.

## Multi-Account Setup

Use AWS provider aliases to onboard several accounts at once:

```hcl
provider "overmind" {}

provider "aws" {
  alias  = "production"
  region = "us-east-1"

  assume_role {
    role_arn = "arn:aws:iam::111111111111:role/terraform"
  }
}

provider "aws" {
  alias  = "staging"
  region = "eu-west-1"

  assume_role {
    role_arn = "arn:aws:iam::222222222222:role/terraform"
  }
}

module "overmind_production" {
  source = "overmindtech/aws-source/overmind"
  name   = "production"

  providers = {
    aws      = aws.production
    overmind = overmind
  }
}

module "overmind_staging" {
  source  = "overmindtech/aws-source/overmind"
  name    = "staging"
  regions = ["eu-west-1"]

  providers = {
    aws      = aws.staging
    overmind = overmind
  }
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

## Importing Existing Sources

If you already created an Overmind AWS source through the UI and want to bring it under Terraform management, you can import it using the source UUID. Find the UUID on the source details page in [Settings > Sources](https://app.overmind.tech/settings/sources).

When using the module:

```shell
terraform import module.overmind_aws_source.overmind_aws_source.this <source-uuid>
```

When using the provider resource directly:

```shell
terraform import overmind_aws_source.example <source-uuid>
```

After importing, run `terraform plan` to verify the state matches your configuration. Terraform will show any drift between the imported resource and your HCL.

Note that importing brings only the Overmind source under Terraform management. If the IAM role was also created outside of Terraform, you will need to import it separately with `terraform import aws_iam_role.overmind <role-name>`.

## Verify Your Source

After `terraform apply` completes:

1. Open [Settings > Sources](https://app.overmind.tech/settings/sources) in the Overmind app.
2. Your new source should appear with a green healthy status within about a minute.
3. Navigate to [Explore](https://app.overmind.tech/explore) to browse discovered resources.

## Registry Links

- **Terraform Registry**: [overmindtech/overmind provider](https://registry.terraform.io/providers/overmindtech/overmind/latest) | [overmindtech/aws-source module](https://registry.terraform.io/modules/overmindtech/aws-source/overmind/latest)
- **OpenTofu Registry**: coming soon

## Troubleshooting

### "Provider not found" during terraform init

Ensure you are running Terraform >= 1.5.0 or OpenTofu >= 1.6.0, and that you have internet access to reach the registry. Run `terraform init -upgrade` to refresh provider caches.

### "Unauthorized" or "invalid API key"

Verify that `OVERMIND_API_KEY` is set and that the key has `sources:write` scope. You can check your API keys in [Settings > API Keys](https://app.overmind.tech/settings/api-keys).

### "Access Denied" creating IAM resources

The AWS credentials used by Terraform need permission to create IAM roles and policies. Verify your credentials have the `iam:CreateRole`, `iam:PutRolePolicy`, and `iam:CreatePolicy` permissions in the target account.

### Source shows as unhealthy after apply

The IAM role may take a few seconds to propagate. Wait one to two minutes and refresh the Sources page. If the source remains unhealthy, verify the role ARN in the AWS console matches the `role_arn` output.

### Destroying resources

`terraform destroy` cleanly removes both the IAM resources in AWS and the Overmind source registration.
