package tfutils

import (
	"context"
	"testing"
)

func TestParseAWSProviders(t *testing.T) {
	providers, files, err := ParseAWSProviders("testdata")
	if err != nil {
		t.Errorf("Error parsing AWS providers: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	if len(providers) != 5 {
		t.Fatalf("Expected 5 providers, got %d", len(providers))
	}

	if providers[1].Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", providers[0].Region)
	}

	if providers[2].AssumeRole.RoleARN != "arn:aws:iam::123456789012:role/ROLE_NAME" {
		t.Errorf("Expected role arn arn:aws:iam::123456789012:role/ROLE_NAME, got %s", providers[2].AssumeRole.RoleARN)
	}

	if providers[2].AssumeRole.SessionName != "SESSION_NAME" {
		t.Errorf("Expected session name SESSION_NAME, got % s", providers[2].AssumeRole.SessionName)
	}

	if providers[2].AssumeRole.ExternalID != "EXTERNAL_ID" {
		t.Errorf("Expected external id EXTERNAL_ID, got %s", providers[2].AssumeRole.ExternalID)
	}

	if providers[3].AccessKey != "access_key" {
		t.Errorf("Expected access key access_key, got %s", providers[3].AccessKey)
	}

	if providers[3].SecretKey != "secret_key" {
		t.Errorf("Expected secret key secret_key, got %s", providers[3].SecretKey)
	}

	if providers[3].Token != "token" {
		t.Errorf("Expected token token, got %s", providers[3].Token)
	}

	if providers[3].Region != "region" {
		t.Errorf("Expected region region, got %s", providers[3].Region)
	}

	if providers[3].CustomCABundle != "testdata/providers.tf" {
		t.Errorf("Expected custom ca bundle testdata/providers.tf, got %s", providers[3].CustomCABundle)
	}

	if providers[3].EC2MetadataServiceEndpoint != "ec2_metadata_service_endpoint" {
		t.Errorf("Expected ec2 metadata service endpoint ec2_metadata_service_endpoint, got %s", providers[3].EC2MetadataServiceEndpoint)
	}

	if providers[3].EC2MetadataServiceEndpointMode != "ipv6" {
		t.Errorf("Expected ec2 metadata service endpoint mode ipv6, got %s", providers[3].EC2MetadataServiceEndpointMode)
	}

	if providers[3].SkipMetadataAPICheck != true {
		t.Errorf("Expected skip metadata api check true, got %t", providers[3].SkipMetadataAPICheck)
	}

	if providers[3].HTTPProxy != "http_proxy" {
		t.Errorf("Expected http proxy http_proxy, got %s", providers[3].HTTPProxy)
	}

	if providers[3].HTTPSProxy != "https_proxy" {
		t.Errorf("Expected https proxy https_proxy, got %s", providers[3].HTTPSProxy)
	}

	if providers[3].NoProxy != "no_proxy" {
		t.Errorf("Expected no proxy no_proxy, got %s", providers[3].NoProxy)
	}

	if providers[3].MaxRetries != 10 {
		t.Errorf("Expected max retries 10, got %d", providers[3].MaxRetries)
	}

	if providers[3].Profile != "profile" {
		t.Errorf("Expected profile profile, got %s", providers[3].Profile)
	}

	if providers[3].RetryMode != "standard" {
		t.Errorf("Expected retry mode standard, got %s", providers[3].RetryMode)
	}

	if len(providers[3].SharedConfigFiles) != 1 {
		t.Errorf("Expected 1 shared config file, got %d", len(providers[3].SharedConfigFiles))
	}

	if providers[3].SharedConfigFiles[0] != "shared_config_files" {
		t.Errorf("Expected shared config file shared_config_files, got %s", providers[3].SharedConfigFiles[0])
	}

	if len(providers[3].SharedCredentialsFiles) != 1 {
		t.Errorf("Expected 1 shared credentials file, got %d", len(providers[3].SharedCredentialsFiles))
	}

	if providers[3].SharedCredentialsFiles[0] != "shared_credentials_files" {
		t.Errorf("Expected shared credentials file shared_credentials_files, got %s", providers[3].SharedCredentialsFiles[0])
	}

	if providers[3].UseDualStackEndpoint != false {
		t.Errorf("Expected use dual stack endpoint false, got %t", providers[3].UseDualStackEndpoint)
	}

	if providers[3].UseFIPSEndpoint != false {
		t.Errorf("Expected use fips endpoint false, got %t", providers[3].UseFIPSEndpoint)
	}

	if providers[3].AssumeRoleWithWebIdentity.RoleARN != "arn:aws:iam::123456789012:role/ROLE_NAME" {
		t.Errorf("Expected role arn arn:aws:iam::123456789012:role/ROLE_NAME, got %s", providers[3].AssumeRoleWithWebIdentity.RoleARN)
	}

	if providers[3].AssumeRoleWithWebIdentity.SessionName != "SESSION_NAME" {
		t.Errorf("Expected session name SESSION_NAME, got %s", providers[3].AssumeRoleWithWebIdentity.SessionName)
	}

	if providers[3].AssumeRoleWithWebIdentity.WebIdentityTokenFile != "/Users/tf_user/secrets/web-identity-token" {
		t.Errorf("Expected web identity token file /Users/tf_user/secrets/web-identity-token, got %s", providers[3].AssumeRoleWithWebIdentity.WebIdentityTokenFile)
	}

	if providers[3].AssumeRoleWithWebIdentity.WebIdentityToken != "web_identity_token" {
		t.Errorf("Expected web identity token web_identity_token, got %s", providers[3].AssumeRoleWithWebIdentity.WebIdentityToken)
	}

	if providers[3].AssumeRoleWithWebIdentity.Duration != "1s" {
		t.Errorf("Expected duration 1s, got %s", providers[3].AssumeRoleWithWebIdentity.Duration)
	}

	if providers[3].AssumeRoleWithWebIdentity.Policy != "policy" {
		t.Errorf("Expected policy policy, got %s", providers[3].AssumeRoleWithWebIdentity.Policy)
	}

	if len(providers[3].AssumeRoleWithWebIdentity.PolicyARNs) != 1 {
		t.Errorf("Expected 1 policy arn, got %d", len(providers[3].AssumeRoleWithWebIdentity.PolicyARNs))
	}

	if providers[3].AssumeRoleWithWebIdentity.PolicyARNs[0] != "policy_arns" {
		t.Errorf("Expected policy arn policy_arns, got %s", providers[3].AssumeRoleWithWebIdentity.PolicyARNs[0])
	}

	if providers[4].Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", providers[4].Region)
	}

	if providers[4].AccessKey != "my-access-key" {
		t.Errorf("Expected access key my-access-key, got %s", providers[4].AccessKey)
	}

	if providers[4].SecretKey != "my-secret-key" {
		t.Errorf("Expected secret key my-secret-key, got %s", providers[4].SecretKey)
	}
}

func TestConfigFromProvider(t *testing.T) {
	// Make sure the providers we have created can all be turned into configs
	// without any issues
	providers, _, err := ParseAWSProviders("testdata/config_from_provider")
	if err != nil {
		t.Fatalf("Error parsing AWS providers: %v", err)
	}

	for _, provider := range providers {
		_, err := ConfigFromProvider(context.Background(), provider)
		if err != nil {
			t.Errorf("Error converting provider to config: %v", err)
		}
	}
}
