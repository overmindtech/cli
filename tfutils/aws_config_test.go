package tfutils

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestParseAWSProviders(t *testing.T) {
	results, err := ParseAWSProviders("testdata", nil)
	if err != nil {
		t.Errorf("Error parsing AWS results: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if results[1].Provider.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", results[0].Provider.Region)
	}

	if results[2].Provider.AssumeRole.RoleARN != "arn:aws:iam::123456789012:role/ROLE_NAME" {
		t.Errorf("Expected role arn arn:aws:iam::123456789012:role/ROLE_NAME, got %s", results[2].Provider.AssumeRole.RoleARN)
	}

	if results[2].Provider.AssumeRole.SessionName != "SESSION_NAME" {
		t.Errorf("Expected session name SESSION_NAME, got % s", results[2].Provider.AssumeRole.SessionName)
	}

	if results[2].Provider.AssumeRole.ExternalID != "EXTERNAL_ID" {
		t.Errorf("Expected external id EXTERNAL_ID, got %s", results[2].Provider.AssumeRole.ExternalID)
	}

	if results[3].Provider.AccessKey != "access_key" {
		t.Errorf("Expected access key access_key, got %s", results[3].Provider.AccessKey)
	}

	if results[3].Provider.SecretKey != "secret_key" {
		t.Errorf("Expected secret key secret_key, got %s", results[3].Provider.SecretKey)
	}

	if results[3].Provider.Token != "token" {
		t.Errorf("Expected token token, got %s", results[3].Provider.Token)
	}

	if results[3].Provider.Region != "region" {
		t.Errorf("Expected region region, got %s", results[3].Provider.Region)
	}

	if results[3].Provider.CustomCABundle != "testdata/providers.tf" {
		t.Errorf("Expected custom ca bundle testdata/providers.tf, got %s", results[3].Provider.CustomCABundle)
	}

	if results[3].Provider.EC2MetadataServiceEndpoint != "ec2_metadata_service_endpoint" {
		t.Errorf("Expected ec2 metadata service endpoint ec2_metadata_service_endpoint, got %s", results[3].Provider.EC2MetadataServiceEndpoint)
	}

	if results[3].Provider.EC2MetadataServiceEndpointMode != "ipv6" {
		t.Errorf("Expected ec2 metadata service endpoint mode ipv6, got %s", results[3].Provider.EC2MetadataServiceEndpointMode)
	}

	if results[3].Provider.SkipMetadataAPICheck != true {
		t.Errorf("Expected skip metadata api check true, got %t", results[3].Provider.SkipMetadataAPICheck)
	}

	if results[3].Provider.HTTPProxy != "http_proxy" {
		t.Errorf("Expected http proxy http_proxy, got %s", results[3].Provider.HTTPProxy)
	}

	if results[3].Provider.HTTPSProxy != "https_proxy" {
		t.Errorf("Expected https proxy https_proxy, got %s", results[3].Provider.HTTPSProxy)
	}

	if results[3].Provider.NoProxy != "no_proxy" {
		t.Errorf("Expected no proxy no_proxy, got %s", results[3].Provider.NoProxy)
	}

	if results[3].Provider.MaxRetries != 10 {
		t.Errorf("Expected max retries 10, got %d", results[3].Provider.MaxRetries)
	}

	if results[3].Provider.Profile != "profile" {
		t.Errorf("Expected profile profile, got %s", results[3].Provider.Profile)
	}

	if results[3].Provider.RetryMode != "standard" {
		t.Errorf("Expected retry mode standard, got %s", results[3].Provider.RetryMode)
	}

	if len(results[3].Provider.SharedConfigFiles) != 1 {
		t.Errorf("Expected 1 shared config file, got %d", len(results[3].Provider.SharedConfigFiles))
	}

	if results[3].Provider.SharedConfigFiles[0] != "shared_config_files" {
		t.Errorf("Expected shared config file shared_config_files, got %s", results[3].Provider.SharedConfigFiles[0])
	}

	if len(results[3].Provider.SharedCredentialsFiles) != 1 {
		t.Errorf("Expected 1 shared credentials file, got %d", len(results[3].Provider.SharedCredentialsFiles))
	}

	if results[3].Provider.SharedCredentialsFiles[0] != "shared_credentials_files" {
		t.Errorf("Expected shared credentials file shared_credentials_files, got %s", results[3].Provider.SharedCredentialsFiles[0])
	}

	if results[3].Provider.UseDualStackEndpoint != false {
		t.Errorf("Expected use dual stack endpoint false, got %t", results[3].Provider.UseDualStackEndpoint)
	}

	if results[3].Provider.UseFIPSEndpoint != false {
		t.Errorf("Expected use fips endpoint false, got %t", results[3].Provider.UseFIPSEndpoint)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.RoleARN != "arn:aws:iam::123456789012:role/ROLE_NAME" {
		t.Errorf("Expected role arn arn:aws:iam::123456789012:role/ROLE_NAME, got %s", results[3].Provider.AssumeRoleWithWebIdentity.RoleARN)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.SessionName != "SESSION_NAME" {
		t.Errorf("Expected session name SESSION_NAME, got %s", results[3].Provider.AssumeRoleWithWebIdentity.SessionName)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.WebIdentityTokenFile != "/Users/tf_user/secrets/web-identity-token" {
		t.Errorf("Expected web identity token file /Users/tf_user/secrets/web-identity-token, got %s", results[3].Provider.AssumeRoleWithWebIdentity.WebIdentityTokenFile)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.WebIdentityToken != "web_identity_token" {
		t.Errorf("Expected web identity token web_identity_token, got %s", results[3].Provider.AssumeRoleWithWebIdentity.WebIdentityToken)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.Duration != "1s" {
		t.Errorf("Expected duration 1s, got %s", results[3].Provider.AssumeRoleWithWebIdentity.Duration)
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.Policy != "policy" {
		t.Errorf("Expected policy policy, got %s", results[3].Provider.AssumeRoleWithWebIdentity.Policy)
	}

	if len(results[3].Provider.AssumeRoleWithWebIdentity.PolicyARNs) != 1 {
		t.Errorf("Expected 1 policy arn, got %d", len(results[3].Provider.AssumeRoleWithWebIdentity.PolicyARNs))
	}

	if results[3].Provider.AssumeRoleWithWebIdentity.PolicyARNs[0] != "policy_arns" {
		t.Errorf("Expected policy arn policy_arns, got %s", results[3].Provider.AssumeRoleWithWebIdentity.PolicyARNs[0])
	}

	if results[4].Provider.Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", results[4].Provider.Region)
	}

	if results[4].Provider.AccessKey != "my-access-key" {
		t.Errorf("Expected access key my-access-key, got %s", results[4].Provider.AccessKey)
	}

	if results[4].Provider.SecretKey != "my-secret-key" {
		t.Errorf("Expected secret key my-secret-key, got %s", results[4].Provider.SecretKey)
	}
}

func TestConfigFromProvider(t *testing.T) {
	// Make sure the providers we have created can all be turned into configs
	// without any issues
	results, err := ParseAWSProviders("testdata/config_from_provider", nil)
	if err != nil {
		t.Fatalf("Error parsing AWS providers: %v", err)
	}

	for _, provider := range results {
		_, err := ConfigFromProvider(context.Background(), *provider.Provider)
		if err != nil {
			t.Errorf("Error converting provider to config: %v", err)
		}
	}
}

func TestParseTFVarsFile(t *testing.T) {
	t.Run("with a good file", func(t *testing.T) {
		evalCtx := hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		}

		err := ParseTFVarsFile("testdata/test_vars.tfvars", &evalCtx)
		if err != nil {
			t.Fatalf("Error parsing TF vars file: %v", err)
		}

		if evalCtx.Variables["simple_string"].Type() != cty.String {
			t.Errorf("Expected simple_string to be a string, got %s", evalCtx.Variables["simple_string"].Type())
		}

		if evalCtx.Variables["simple_string"].AsString() != "example_string" {
			t.Errorf("Expected simple_string to be example_string, got %s", evalCtx.Variables["simple_string"].AsString())
		}

		if evalCtx.Variables["example_number"].Type() != cty.Number {
			t.Errorf("Expected example_number to be a number, got %s", evalCtx.Variables["example_number"].Type())
		}

		if evalCtx.Variables["example_number"].AsBigFloat().String() != "42" {
			t.Errorf("Expected example_number to be 42, got %s", evalCtx.Variables["example_number"].AsBigFloat().String())
		}

		if evalCtx.Variables["example_boolean"].Type() != cty.Bool {
			t.Errorf("Expected example_boolean to be a bool, got %s", evalCtx.Variables["example_boolean"].Type())
		}

		if values := evalCtx.Variables["example_list"].AsValueSlice(); len(values) == 3 {
			if values[0].AsString() != "item1" {
				t.Errorf("Expected first item to be item1, got %s", values[0].AsString())
			}
		} else {
			t.Errorf("Expected example_list to have 3 elements, got %d", len(values))
		}

		if m := evalCtx.Variables["example_map"].AsValueMap(); len(m) == 2 {
			if m["key1"].AsString() != "value1" {
				t.Errorf("Expected key1 to be value1, got %s", m["key1"].AsString())
			}
		} else {
			t.Errorf("Expected example_map to have 2 elements, got %d", len(m))
		}
	})

	t.Run("with a file that doesn't exist", func(t *testing.T) {
		evalCtx := hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		}

		err := ParseTFVarsFile("testdata/nonexistent.tfvars", &evalCtx)
		if err == nil {
			t.Fatalf("Expected error parsing nonexistent file, got nil")
		}
	})

	t.Run("with a file that has invalid syntax", func(t *testing.T) {
		evalCtx := hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		}

		err := ParseTFVarsFile("testdata/invalid_vars.tfvars", &evalCtx)
		if err == nil {
			t.Fatalf("Expected error parsing invalid syntax file, got nil")
		}
	})
}

func TestParseTFVarsJSONFile(t *testing.T) {
	t.Run("with a good file", func(t *testing.T) {
		evalCtx := hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		}

		err := ParseTFVarsJSONFile("testdata/tfvars.json", &evalCtx)
		if err != nil {
			t.Fatalf("Error parsing TF vars file: %v", err)
		}

		if evalCtx.Variables["string"].Type() != cty.String {
			t.Errorf("Expected string to be a string, got %s", evalCtx.Variables["string"].Type())
		}

		if evalCtx.Variables["string"].AsString() != "example_string" {
			t.Errorf("Expected string to be example_string, got %s", evalCtx.Variables["string"].AsString())
		}

		if values := evalCtx.Variables["list"].AsValueSlice(); len(values) == 2 {
			if values[0].AsString() != "item1" {
				t.Errorf("Expected first item to be item1, got %s", values[0].AsString())
			}
		} else {
			t.Errorf("Expected list to have 2 elements, got %d", len(values))
		}
	})

	t.Run("with a file that doesn't exist", func(t *testing.T) {
		evalCtx := hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		}

		err := ParseTFVarsJSONFile("testdata/nonexistent.json", &evalCtx)
		if err == nil {
			t.Fatalf("Expected error parsing nonexistent file, got nil")
		}
	})
}

func TestParseFlagValue(t *testing.T) {
	// There are a number of ways to supply ags, for example:
	//
	// terraform apply
	// terraform apply -var "image_id=ami-abc123"
	// terraform apply -var 'name=value'
	// terraform apply -var='image_id_list=["ami-abc123","ami-def456"]' -var="instance_type=t2.micro"
	// terraform apply -var='image_id_map={"us-east-1":"ami-abc123","us-east-2":"ami-def456"}'

	tests := []struct {
		Name  string
		Value string
	}{
		{
			Name:  "with =",
			Value: "image_id=ami-abc123",
		},
		{
			Name:  "with a space",
			Value: "image_id=ami-abc123",
		},
		{
			Name:  "with a list",
			Value: "image_id_list=[\"ami-abc123\",\"ami-def456\"]",
		},
		{
			Name:  "with a map",
			Value: "image_id_map={\"us-east-1\":\"ami-abc123\",\"us-east-2\":\"ami-def456\"}",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			evalCtx := hcl.EvalContext{
				Variables: make(map[string]cty.Value),
			}

			err := ParseFlagValue(test.Value, &evalCtx)
			if err != nil {
				t.Fatalf("Error parsing vars args: %v", err)
			}
		})
	}
}

func TestParseVarsArgs(t *testing.T) {
	tests := []struct {
		Name string
		Args []string
	}{
		{
			Name: "with a single var",
			Args: []string{"-var", "image_id=ami-abc123"},
		},
		{
			Name: "with multiple vars",
			Args: []string{"-var", "image_id=ami-abc123", "-var", "instance_type=t2.micro"},
		},
		{
			Name: "with a vars file",
			Args: []string{"-var-file", "testdata/test_vars.tfvars"},
		},
		{
			Name: "with a vars json file",
			Args: []string{"-var-file", "testdata/tfvars.json"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			evalCtx := hcl.EvalContext{
				Variables: make(map[string]cty.Value),
			}

			err := ParseVarsArgs(test.Args, &evalCtx)
			if err != nil {
				t.Fatalf("Error parsing vars args: %v", err)
			}
		})
	}
}

func TestLoadEvalContext(t *testing.T) {
	args := []string{
		"plan",
		"-var", "image_id=args",
		"-var", "instance_type=t2.micro",
		"-var-file", "testdata/tfvars.json",
		"-var-file=testdata/test_vars.tfvars",
	}

	env := []string{
		"TF_VAR_image_id=environment",
	}

	evalCtx, err := LoadEvalContext(args, env)
	if err != nil {
		t.Fatal(err)
	}

	if evalCtx.Variables["image_id"].AsString() != "args" {
		t.Errorf("Expected image_id to be args, got %s", evalCtx.Variables["image_id"].AsString())
	}
}
