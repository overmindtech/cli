terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    overmind = {
      source  = "overmindtech/overmind"
      version = ">= 0.1.0"
    }
  }
}
