# This is a very simple example to deploy a few cheap resources into AWS to test the new `terraform plan` and `terraform apply` subcommands.

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.56"
    }
  }
}

provider "aws" {}
provider "aws" {
  alias  = "aliased"
  region = "us-east-1"
}

variable "bucket_postfix" {
  type        = string
  description = "The prefix to apply to the bucket name."
  default     = "test"
}

module "bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "~> 4.0"

  bucket_prefix = "cli-test${var.bucket_postfix}"

  control_object_ownership = true
  object_ownership         = "BucketOwnerEnforced"
  block_public_policy      = true
  block_public_acls        = true
  ignore_public_acls       = true
  restrict_public_buckets  = true
}
