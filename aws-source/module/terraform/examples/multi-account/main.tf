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

output "production_role_arn" {
  value = module.overmind_production.role_arn
}

output "staging_role_arn" {
  value = module.overmind_staging.role_arn
}
