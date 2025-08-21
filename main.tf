# This is a very simple example to deploy a few cheap resources into AWS to test the new `terraform plan` and `terraform apply` subcommands.

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.56"
    }
    google = {
      source  = "hashicorp/google"
      version = ">= 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
  }
}

provider "aws" {}
provider "aws" {
  alias  = "aliased"
  region = "us-east-1"
}

provider "google" {
  project = "overmind-demo"
  region  = "us-central1"
}

provider "google" {
  alias   = "west"
  project = "overmind-demo-west"
  region  = "us-west1"
  zone    = "us-west1-a"
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

# Simple GCP storage buckets for testing multiple providers
resource "google_storage_bucket" "test" {
  name     = "cli-test-${var.bucket_postfix}-${random_id.bucket_suffix.hex}"
  location = "US"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
}

resource "google_storage_bucket" "test_west" {
  provider = google.west
  name     = "cli-test-west-${var.bucket_postfix}-${random_id.bucket_suffix.hex}"
  location = "US-WEST1"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
}

resource "random_id" "bucket_suffix" {
  byte_length = 8
}
