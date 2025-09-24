package tfutils

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestParseAWSProviders(t *testing.T) {
	t.Run("non-recursive only current directory", func(t *testing.T) {
		results, err := ParseAWSProviders("testdata", nil, false)
		if err != nil {
			t.Fatalf("Error parsing AWS results: %v", err)
		}
		// Expect 3 providers from providers.tf only
		if len(results) != 3 {
			t.Fatalf("Expected 3 results (non-recursive), got %d", len(results))
		}
	})

	t.Run("recursive finds providers in subdirectories", func(t *testing.T) {
		results, err := ParseAWSProviders("testdata", nil, true)
		if err != nil {
			t.Fatalf("Error parsing AWS results: %v", err)
		}

		// Expect 5 results when recursive:
		// - 3 from providers.tf
		// - 1 from subfolder/more_providers.tf
		// - 1 from config_from_provider/test.tf
		if len(results) != 5 {
			t.Fatalf("Expected 5 results (recursive), got %d", len(results))
		}

		// Count providers by their characteristics to make test order-independent
		var foundUsEast1, foundAssumeRole, foundEverything, foundSubdir, foundConfigTest int

		for _, result := range results {
			if result.Provider == nil {
				continue
			}

			if result.Provider.Region == "us-east-1" && result.Provider.Alias == "" {
				foundUsEast1++
			}
			if result.Provider.Alias == "assume_role" && result.Provider.AssumeRole != nil {
				foundAssumeRole++
				if result.Provider.AssumeRole.RoleARN != "arn:aws:iam::123456789012:role/ROLE_NAME" {
					t.Errorf("Expected role arn arn:aws:iam::123456789012:role/ROLE_NAME, got %s", result.Provider.AssumeRole.RoleARN)
				}
				if result.Provider.AssumeRole.SessionName != "SESSION_NAME" {
					t.Errorf("Expected session name SESSION_NAME, got %s", result.Provider.AssumeRole.SessionName)
				}
				if result.Provider.AssumeRole.ExternalID != "EXTERNAL_ID" {
					t.Errorf("Expected external id EXTERNAL_ID, got %s", result.Provider.AssumeRole.ExternalID)
				}
			}
			if result.Provider.Alias == "everything" {
				foundEverything++
				if strings.Contains(result.FilePath, "config_from_provider") {
					foundConfigTest++
				}
			}
			if result.Provider.Alias == "subdir" && result.Provider.Region == "us-west-2" {
				foundSubdir++
				if !strings.Contains(result.FilePath, "subfolder") {
					t.Errorf("Expected subdir provider to be in subfolder, got path: %s", result.FilePath)
				}
			}
		}

		if foundUsEast1 != 1 {
			t.Errorf("Expected to find 1 us-east-1 provider, found %d", foundUsEast1)
		}
		if foundAssumeRole != 1 {
			t.Errorf("Expected to find 1 assume_role provider, found %d", foundAssumeRole)
		}
		if foundEverything != 2 { // One from providers.tf and one from config_from_provider/test.tf
			t.Errorf("Expected to find 2 'everything' providers, found %d", foundEverything)
		}
		if foundSubdir != 1 {
			t.Errorf("Expected to find 1 subdir provider, found %d", foundSubdir)
		}
		if foundConfigTest != 1 {
			t.Errorf("Expected to find 1 provider in config_from_provider, found %d", foundConfigTest)
		}
	})
}

func TestConfigFromProvider(t *testing.T) {
	t.Setenv("AWS_PROFILE", "")
	// Make sure the providers we have created can all be turned into configs
	// without any issues
	results, err := ParseAWSProviders("testdata/config_from_provider", nil, false)
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

		if !evalCtx.Variables["var"].Type().IsObjectType() {
			t.Errorf("Expected var to be an object, got %s", evalCtx.Variables["var"].Type())
		}

		variables := evalCtx.Variables["var"].AsValueMap()

		if variables["simple_string"].Type() != cty.String {
			t.Errorf("Expected simple_string to be a string, got %s", variables["simple_string"].Type())
		}

		if variables["simple_string"].AsString() != "example_string" {
			t.Errorf("Expected simple_string to be example_string, got %s", variables["simple_string"].AsString())
		}

		if variables["example_number"].Type() != cty.Number {
			t.Errorf("Expected example_number to be a number, got %s", variables["example_number"].Type())
		}

		if variables["example_number"].AsBigFloat().String() != "42" {
			t.Errorf("Expected example_number to be 42, got %s", variables["example_number"].AsBigFloat().String())
		}

		if variables["example_boolean"].Type() != cty.Bool {
			t.Errorf("Expected example_boolean to be a bool, got %s", variables["example_boolean"].Type())
		}

		if values := variables["example_list"].AsValueSlice(); len(values) == 3 {
			if values[0].AsString() != "item1" {
				t.Errorf("Expected first item to be item1, got %s", values[0].AsString())
			}
		} else {
			t.Errorf("Expected example_list to have 3 elements, got %d", len(values))
		}

		if m := variables["example_map"].AsValueMap(); len(m) == 2 {
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

		if !evalCtx.Variables["var"].Type().IsObjectType() {
			t.Errorf("Expected var to be an object, got %s", evalCtx.Variables["var"].Type())
		}

		variables := evalCtx.Variables["var"].AsValueMap()

		if variables["string"].Type() != cty.String {
			t.Errorf("Expected string to be a string, got %s", variables["string"].Type())
		}

		if variables["string"].AsString() != "example_string" {
			t.Errorf("Expected string to be example_string, got %s", variables["string"].AsString())
		}

		if values := variables["list"].AsValueSlice(); len(values) == 2 {
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
		"--var", "instance_type=t2.micro",
		"-var-file", "testdata/tfvars.json",
		"-var-file=testdata/test_vars.tfvars",
	}

	env := []string{
		"TF_VAR_something=else",
		"TF_VAR_image_id=environment",
	}

	evalCtx, err := LoadEvalContext(args, env)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(evalCtx)

	variables := evalCtx.Variables["var"].AsValueMap()

	if variables["instance_type"].AsString() != "t2.micro" {
		t.Errorf("Expected instance_type to be t2.micro, got %s", variables["instance_type"].AsString())
	}
	if variables["something"].AsString() != "else" {
		t.Errorf("Expected something to be else, got %s", variables["something"].AsString())
	}
	if variables["image_id"].AsString() != "args" {
		t.Errorf("Expected image_id to be args, got %s", variables["image_id"].AsString())
	}
}

func TestParseAWSProvidersWithSubmodules(t *testing.T) {
	// Test parsing providers in nested modules
	if _, err := os.Stat("testdata_nested_modules"); err != nil {
		t.Skip("skipping: test fixture 'testdata_nested_modules' not present")
	}
	results, err := ParseAWSProviders("testdata_nested_modules", nil, true)
	if err != nil {
		t.Errorf("Error parsing AWS providers in nested modules: %v", err)
	}

	// We expect 4 providers:
	// 1. Root module (us-east-1)
	// 2. VPC module (us-west-2)
	// 3. EC2 module (eu-west-1 with assume_role)
	// 4. Nested submodule (ap-southeast-1)
	if len(results) != 4 {
		t.Fatalf("Expected 4 providers in nested modules, got %d", len(results))
	}

	// Map to track found providers by region
	providersByRegion := make(map[string]*ProviderResult)
	for i := range results {
		result := &results[i]
		if result.Error != nil {
			t.Errorf("Error in result for file %s: %v", result.FilePath, result.Error)
			continue
		}
		if result.Provider != nil {
			providersByRegion[result.Provider.Region] = result
		}
	}

	// Verify root provider
	if rootProvider, ok := providersByRegion["us-east-1"]; ok {
		if !strings.Contains(rootProvider.FilePath, "main.tf") {
			t.Errorf("Expected root provider to be in main.tf, got %s", rootProvider.FilePath)
		}
	} else {
		t.Errorf("Expected to find provider with region us-east-1")
	}

	// Verify VPC module provider
	if vpcProvider, ok := providersByRegion["us-west-2"]; ok {
		if vpcProvider.Provider.Alias != "vpc_module" {
			t.Errorf("Expected VPC provider alias to be vpc_module, got %s", vpcProvider.Provider.Alias)
		}
		if !strings.Contains(vpcProvider.FilePath, "modules/vpc/providers.tf") {
			t.Errorf("Expected VPC provider to be in modules/vpc/providers.tf, got %s", vpcProvider.FilePath)
		}
	} else {
		t.Errorf("Expected to find provider with region us-west-2")
	}

	// Verify EC2 module provider with assume role
	if ec2Provider, ok := providersByRegion["eu-west-1"]; ok {
		if ec2Provider.Provider.Alias != "ec2_module" {
			t.Errorf("Expected EC2 provider alias to be ec2_module, got %s", ec2Provider.Provider.Alias)
		}
		if ec2Provider.Provider.AssumeRole == nil {
			t.Errorf("Expected EC2 provider to have assume_role configuration")
		} else if ec2Provider.Provider.AssumeRole.RoleARN != "arn:aws:iam::987654321098:role/EC2ModuleRole" {
			t.Errorf("Expected EC2 provider role ARN to be arn:aws:iam::987654321098:role/EC2ModuleRole, got %s", ec2Provider.Provider.AssumeRole.RoleARN)
		}
		if !strings.Contains(ec2Provider.FilePath, "modules/ec2/providers.tf") {
			t.Errorf("Expected EC2 provider to be in modules/ec2/providers.tf, got %s", ec2Provider.FilePath)
		}
	} else {
		t.Errorf("Expected to find provider with region eu-west-1")
	}

	// Verify deeply nested provider
	if nestedProvider, ok := providersByRegion["ap-southeast-1"]; ok {
		if nestedProvider.Provider.Alias != "nested_provider" {
			t.Errorf("Expected nested provider alias to be nested_provider, got %s", nestedProvider.Provider.Alias)
		}
		if nestedProvider.Provider.AccessKey != "nested-access-key" {
			t.Errorf("Expected nested provider access key to be nested-access-key, got %s", nestedProvider.Provider.AccessKey)
		}
		if !strings.Contains(nestedProvider.FilePath, "modules/ec2/nested_submodule/providers.tf") {
			t.Errorf("Expected nested provider to be in modules/ec2/nested_submodule/providers.tf, got %s", nestedProvider.FilePath)
		}
	} else {
		t.Errorf("Expected to find provider with region ap-southeast-1")
	}
}

func TestParseAWSProviders_RecursiveNestedExample(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	mustWrite := func(relPath, content string) {
		full := filepath.Join(tempDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir failed for %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write failed for %s: %v", full, err)
		}
	}

	// Root provider
	mustWrite("providers.tf", `
provider "aws" {
  region = "us-east-1"
  default_tags {
    tags = { Environment = "production", ManagedBy = "terraform" }
  }
}
`)

	// modules/networking provider
	mustWrite("modules/networking/providers.tf", `
provider "aws" {
  alias  = "networking"
  region = "us-west-2"
  default_tags {
    tags = { Module = "networking", Team = "infrastructure" }
  }
}
`)

	// modules/networking/vpc provider with assume_role
	mustWrite("modules/networking/vpc/providers.tf", `
provider "aws" {
  alias  = "vpc_endpoints"
  region = "us-west-2"
  assume_role {
    role_arn     = "arn:aws:iam::123456789012:role/VPCEndpointManager"
    session_name = "vpc-endpoint-management"
  }
}
`)

	// modules/compute providers (two providers)
	mustWrite("modules/compute/providers.tf", `
provider "aws" {
  alias  = "compute"
  region = "eu-west-1"
  default_tags { tags = { Module = "compute", Team = "platform" } }
}
provider "aws" {
  alias  = "shared_resources"
  region = "eu-west-1"
  assume_role { role_arn = "arn:aws:iam::987654321098:role/SharedResourceAccess" }
}
`)

	// modules/compute/eks provider with assume_role and external_id
	mustWrite("modules/compute/eks/providers.tf", `
provider "aws" {
  alias  = "eks_admin"
  region = "eu-west-1"
  assume_role {
    role_arn     = "arn:aws:iam::123456789012:role/EKSClusterAdmin"
    session_name = "eks-cluster-management"
    external_id  = "eks-external-id"
  }
}
`)

	results, err := ParseAWSProviders(tempDir, nil, true)
	if err != nil {
		t.Fatalf("ParseAWSProviders recursive failed: %v", err)
	}

	if len(results) != 6 {
		t.Fatalf("Expected 6 providers discovered, got %d", len(results))
	}

	// Validate presence and key attributes
	found := map[string]bool{}
	for _, r := range results {
		if r.Provider == nil {
			continue
		}
		key := r.Provider.Region + ":" + r.Provider.Alias
		found[key] = true

		// Check assume_role details where expected
		switch r.Provider.Alias {
		case "vpc_endpoints":
			if r.Provider.AssumeRole == nil || r.Provider.AssumeRole.RoleARN != "arn:aws:iam::123456789012:role/VPCEndpointManager" {
				t.Errorf("vpc_endpoints provider missing/incorrect assume_role")
			}
		case "shared_resources":
			if r.Provider.AssumeRole == nil || r.Provider.AssumeRole.RoleARN != "arn:aws:iam::987654321098:role/SharedResourceAccess" {
				t.Errorf("shared_resources provider missing/incorrect assume_role")
			}
		case "eks_admin":
			if r.Provider.AssumeRole == nil || r.Provider.AssumeRole.RoleARN != "arn:aws:iam::123456789012:role/EKSClusterAdmin" {
				t.Errorf("eks_admin provider missing/incorrect assume_role")
			}
		}
	}

	// Expect these specific providers
	expectedKeys := []string{
		"us-east-1:", // root
		"us-west-2:networking",
		"us-west-2:vpc_endpoints",
		"eu-west-1:compute",
		"eu-west-1:shared_resources",
		"eu-west-1:eks_admin",
	}
	for _, k := range expectedKeys {
		if !found[k] {
			t.Errorf("Expected provider %s not found", k)
		}
	}
}
