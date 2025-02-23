terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }

  required_version = ">= 1.2.0"
}

// Provider that should be ignored
provider "google" {
  project = "acme-app"
  region  = "us-central1"
}

// This should also be ignored
variable "image_id" {
  type = string
}

// This should be ignored too
resource "aws_instance" "app_server" {
  ami           = "ami-830c94e3"
  instance_type = "t2.micro"

  tags = {
    Name = "ExampleAppServerInstance"
  }
}

# Example kube provider using data and functions which we don't support reading
provider "kubernetes" {
  host  = data.aws_eks_cluster.core_eks.endpoint
  token = data.aws_eks_cluster_auth.core_eks.token
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias = "assume_role"

  assume_role {
    role_arn     = "arn:aws:iam::123456789012:role/ROLE_NAME"
    session_name = "SESSION_NAME"
    external_id  = "EXTERNAL_ID"
  }
}

provider "aws" {
  alias                              = "everything"
  access_key                         = "access_key"
  secret_key                         = "secret_key"
  token                              = "token"
  region                             = "region"
  custom_ca_bundle                   = "testdata/providers.tf"
  ec2_metadata_service_endpoint      = "ec2_metadata_service_endpoint"
  ec2_metadata_service_endpoint_mode = "ipv6"
  skip_metadata_api_check            = true
  http_proxy                         = "http_proxy"
  https_proxy                        = "https_proxy"
  no_proxy                           = "no_proxy"
  max_retries                        = 10
  profile                            = "profile"
  retry_mode                         = "standard"
  shared_config_files                = ["shared_config_files"]
  shared_credentials_files           = ["shared_credentials_files"]
  s3_us_east_1_regional_endpoint     = "s3_us_east_1_regional_endpoint"
  use_dualstack_endpoint             = false
  use_fips_endpoint                  = false

  assume_role {
    role_arn     = "arn:aws:iam::123456789012:role/ROLE_NAME"
    session_name = "SESSION_NAME"
    external_id  = "EXTERNAL_ID"
    duration     = "1s"
    policy       = "policy"
    policy_arns  = ["policy_arns"]
    tags = {
      key = "value"
    }
  }

  assume_role_with_web_identity {
    role_arn                = "arn:aws:iam::123456789012:role/ROLE_NAME"
    session_name            = "SESSION_NAME"
    web_identity_token_file = "/Users/tf_user/secrets/web-identity-token"
    web_identity_token      = "web_identity_token"
    duration                = "1s"
    policy                  = "policy"
    policy_arns             = ["policy_arns"]
  }

}
