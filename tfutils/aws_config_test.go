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

	if results[0].Provider.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", results[0].Provider.Region)
	}

	if results[1].Provider.Region != "" {
		t.Errorf("Expected region to be empty, got %s", results[1].Provider.Region)
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
}

func TestConfigFromProvider(t *testing.T) {
	t.Setenv("AWS_PROFILE", "")
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
