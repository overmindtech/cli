# This exists to test the ConfigFromProvider method, we have to omit a few
# things since when we are constructing the AWS config it actually does real
# validation like making sure a profile exists in the shared config files, etc.
# So we have to omit those fields in the test file.
provider "aws" {
  alias                              = "everything"
  access_key                         = "access_key"
  secret_key                         = "secret_key"
  token                              = "token"
  region                             = "region"
  custom_ca_bundle                   = "testdata/config_from_provider/ca-bundle.crt"
  ec2_metadata_service_endpoint      = "ec2_metadata_service_endpoint"
  ec2_metadata_service_endpoint_mode = "ipv6"
  skip_metadata_api_check            = true
  http_proxy                         = "http_proxy"
  https_proxy                        = "https_proxy"
  no_proxy                           = "no_proxy"
  max_retries                        = 10
#   profile                            = "profile"
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