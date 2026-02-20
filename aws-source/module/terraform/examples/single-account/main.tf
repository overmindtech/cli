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
