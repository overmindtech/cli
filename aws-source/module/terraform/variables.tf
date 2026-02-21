variable "name" {
  type        = string
  description = "Descriptive name for the source in Overmind."
}

variable "regions" {
  type = list(string)
  default = [
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2",
    "ap-south-1",
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-northeast-3",
    "ap-southeast-1",
    "ap-southeast-2",
    "ca-central-1",
    "eu-central-1",
    "eu-west-1",
    "eu-west-2",
    "eu-west-3",
    "eu-north-1",
    "sa-east-1",
  ]
  description = "AWS regions to discover. Defaults to all non-opt-in regions."
}

variable "role_name" {
  type        = string
  default     = "overmind-read-only"
  description = "Name for the IAM role created in this account."
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags to apply to IAM resources."
}

variable "overmind_aws_account_id" {
  type        = string
  default     = "942836531449"
  description = "Internal override for the Overmind AWS account that runs source pods. Do not change this unless you are an Overmind engineer deploying to a non-production environment. All customers should use the default."
}
