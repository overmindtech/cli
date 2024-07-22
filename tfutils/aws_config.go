package tfutils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"golang.org/x/net/http/httpproxy"
)

// This struct allows us to parse any HCL file that contains an AWS provider
// using the gohcl library.
type ProviderFile struct {
	Providers []AWSProvider `hcl:"provider,block"`

	// Throw any additional stuff into here so it doesn't fail
	Remain hcl.Body `hcl:",remain"`
}

// This struct represents an AWS provider block in a terraform file. It is
// intended to be used with the gohcl library to parse HCL files.
//
// The fields are based on the AWS provider configuration documentation:
// https://registry.terraform.io/providers/hashicorp/aws/latest/docs#provider-configuration
type AWSProvider struct {
	Name                           string   `hcl:"name,label"`
	AccessKey                      string   `hcl:"access_key,optional"`
	SecretKey                      string   `hcl:"secret_key,optional"`
	Token                          string   `hcl:"token,optional"`
	Region                         string   `hcl:"region,optional"`
	CustomCABundle                 string   `hcl:"custom_ca_bundle,optional"`
	EC2MetadataServiceEndpoint     string   `hcl:"ec2_metadata_service_endpoint,optional"`
	EC2MetadataServiceEndpointMode string   `hcl:"ec2_metadata_service_endpoint_mode,optional"`
	SkipMetadataAPICheck           bool     `hcl:"skip_metadata_api_check,optional"`
	HTTPProxy                      string   `hcl:"http_proxy,optional"`
	HTTPSProxy                     string   `hcl:"https_proxy,optional"`
	NoProxy                        string   `hcl:"no_proxy,optional"`
	MaxRetries                     int      `hcl:"max_retries,optional"`
	Profile                        string   `hcl:"profile,optional"`
	RetryMode                      string   `hcl:"retry_mode,optional"`
	SharedConfigFiles              []string `hcl:"shared_config_files,optional"`
	SharedCredentialsFiles         []string `hcl:"shared_credentials_files,optional"`
	UseDualStackEndpoint           bool     `hcl:"use_dualstack_endpoint,optional"`
	UseFIPSEndpoint                bool     `hcl:"use_fips_endpoint,optional"`

	AssumeRole                *AssumeRole                `hcl:"assume_role,block"`
	AssumeRoleWithWebIdentity *AssumeRoleWithWebIdentity `hcl:"assume_role_with_web_identity,block"`

	// Throw any additional stuff into here so it doesn't fail
	Remain hcl.Body `hcl:",remain"`
}

// Fields that are used for assuming a role, see:
// https://registry.terraform.io/providers/hashicorp/aws/latest/docs#assuming-an-iam-role
type AssumeRole struct {
	Duration          string            `hcl:"duration,optional"`
	ExternalID        string            `hcl:"external_id,optional"`
	Policy            string            `hcl:"policy,optional"`
	PolicyARNs        []string          `hcl:"policy_arns,optional"`
	RoleARN           string            `hcl:"role_arn,optional"`
	SessionName       string            `hcl:"session_name,optional"`
	SourceIdentity    string            `hcl:"source_identity,optional"`
	Tags              map[string]string `hcl:"tags,optional"`
	TransitiveTagKeys []string          `hcl:"transitive_tag_keys,optional"`

	// Throw any additional stuff into here so it doesn't fail
	Remain hcl.Body `hcl:",remain"`
}

// Fields that are used for assuming a role with web identity, see:
// https://registry.terraform.io/providers/hashicorp/aws/latest/docs#assuming-an-iam-role-using-a-web-identity
type AssumeRoleWithWebIdentity struct {
	Duration             string   `hcl:"duration,optional"`
	Policy               string   `hcl:"policy,optional"`
	PolicyARNs           []string `hcl:"policy_arns,optional"`
	RoleARN              string   `hcl:"role_arn,optional"`
	SessionName          string   `hcl:"session_name,optional"`
	WebIdentityToken     string   `hcl:"web_identity_token,optional"`
	WebIdentityTokenFile string   `hcl:"web_identity_token_file,optional"`

	// Throw any additional stuff into here so it doesn't fail
	Remain hcl.Body `hcl:",remain"`
}

// Loads the eval context in the same way that Terraform does, this means it
// supports TF_VAR_* environment variables, terraform.tfvars,
// terraform.tfvars.json, *.auto.tfvars, and *.auto.tfvars.json files, and -var
// and -var-file arguments. These are processed in the order that Terraform uses
// and should result in the same set of variables being loaded.
//
// The args parameter should contain the raw arguments that were passed to
// terraform. This includes: -var and -var-file arguments, and should be passed
// as a list of strings.
//
// The env parameter should contain the environment variables that were present
// when Terraform was run. These should be passed as a []strings (from
// `os.Environ()`), variables beginning with TF_VAR_ will be used.
func LoadEvalContext(args []string, env []string) (*hcl.EvalContext, error) {
	// Note that Terraform has a hierarchy of variable sources, which we need
	// to respect, with later sources taking precedence over earlier ones:
	//
	// * Environment variables
	// * The terraform.tfvars file, if present.
	// * The terraform.tfvars.json file, if present.
	// * Any *.auto.tfvars or *.auto.tfvars.json files, processed in lexical
	//   order of their filenames.
	// * Any -var and -var-file options on the command line, in the order they
	//   are provided. (This includes variables set by an HCP Terraform workspace.)
	evalCtx := hcl.EvalContext{
		Variables: make(map[string]cty.Value),
	}

	// Parse environment variables. Note that if a root module variable uses a
	// type constraint to require a complex value (list, set, map, object, or
	// tuple), Terraform will instead attempt to parse its value using the same
	// syntax used within variable definitions files, which requires careful
	// attention to the string escaping rules in your shell:
	//
	// ```shell
	// export TF_VAR_availability_zone_names='["us-west-1b","us-west-1d"]'
	// ```
	//
	for _, envVar := range env {
		// If the key starts with TF_VAR_, we need to strip that off, and we
		// also want to filter on only these variables
		if strings.HasPrefix(envVar, "TF_VAR_") {
			err := ParseFlagValue(envVar[7:], &evalCtx)
			if err != nil {
				return nil, err
			}
		} else {
			continue
		}
	}

	// Parse the terraform.tfvars file, if present.
	if _, err := os.Stat("terraform.tfvars"); err == nil {
		// Parse the HCL file
		err = ParseTFVarsFile("terraform.tfvars", &evalCtx)
		if err != nil {
			return nil, err
		}
	}

	// Parse the terraform.tfvars.json file, if present.
	if _, err := os.Stat("terraform.tfvars.json"); err == nil {
		// Parse the JSON file
		err = ParseTFVarsJSONFile("terraform.tfvars.json", &evalCtx)
		if err != nil {
			return nil, err
		}
	}

	// Parse *.auto.tfvars or *.auto.tfvars.json files, processed in lexical
	// order of their filenames.
	matches, _ := filepath.Glob("*.auto.tfvars")
	for _, file := range matches {
		// Parse the HCL file
		err := ParseTFVarsFile(file, &evalCtx)
		if err != nil {
			return nil, err
		}
	}

	matches, _ = filepath.Glob("*.auto.tfvars.json")
	for _, file := range matches {
		// Parse the JSON file
		err := ParseTFVarsJSONFile(file, &evalCtx)
		if err != nil {
			return nil, err
		}
	}

	// Parse vars from args, this means the var files and raw vars, in the order
	// they are provided
	err := ParseVarsArgs(args, &evalCtx)
	if err != nil {
		return nil, err
	}

	return &evalCtx, nil
}

// Parses a given TF Vars file into the given eval context
func ParseTFVarsFile(file string, dest *hcl.EvalContext) error {
	// Read the file
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading terraform vars file: %w", err)
	}

	// Parse the HCL file
	parser := hclparse.NewParser()
	parsedFile, diag := parser.ParseHCL(b, file)
	if diag.HasErrors() {
		return fmt.Errorf("error parsing terraform vars file: %w", diag)
	}

	// Decode the body
	var vars map[string]cty.Value
	diag = gohcl.DecodeBody(parsedFile.Body, nil, &vars)
	if diag.HasErrors() {
		return fmt.Errorf("error decoding terraform vars file: %w", diag)
	}

	// Merge the vars into the eval context
	for k, v := range vars {
		dest.Variables[k] = v
	}

	return nil
}

// Parses a given TF Vars JSON file into the given eval context. In this each
// key becomes a variable as par the Hashicorp docs:
// https://developer.hashicorp.com/terraform/language/values/variables#variable-definitions-tfvars-files
func ParseTFVarsJSONFile(file string, dest *hcl.EvalContext) error {
	// Read the file
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading terraform vars file: %w", err)
	}

	// Read the type structure form the file
	ctyType, err := ctyjson.ImpliedType(b)
	if err != nil {
		return fmt.Errorf("error unmarshalling terraform vars file: %w", err)
	}

	// Unmarshal the values
	ctyValue, err := ctyjson.Unmarshal(b, ctyType)
	if err != nil {
		return fmt.Errorf("error unmarshalling terraform vars file: %w", err)
	}

	// Extract the variables
	for k, v := range ctyValue.AsValueMap() {
		dest.Variables[k] = v
	}

	return nil
}

// Parses either a `json` or `tfvars` formatted vars file ands adds these
// variables to the context
func ParseVarsFile(path string, dest *hcl.EvalContext) error {
	switch {
	case strings.HasSuffix(path, ".json"):
		return ParseTFVarsJSONFile(path, dest)
	case strings.HasSuffix(path, ".tfvars"):
		return ParseTFVarsFile(path, dest)
	default:
		return fmt.Errorf("unsupported vars file format: %s", path)
	}
}

// Parses the os.Args for -var and -var-file arguments and adds them to the eval
// context.
func ParseVarsArgs(args []string, dest *hcl.EvalContext) error {
	// We are going to parse the whole argument as HCL here since you can
	// include arrays, maps etc.
	for i, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-var="):
			err := ParseFlagValue(arg[5:], dest)
			if err != nil {
				return err
			}
		case arg == "-var":
			// If the flag is just -var, we need to use the next arg as the value
			// and skip this one
			if i+1 < len(args) {
				err := ParseFlagValue(args[i+1], dest)
				if err != nil {
					return err
				}
			} else {
				continue
			}
		case strings.HasPrefix(arg, "-var-file="):
			err := ParseVarsFile(arg[10:], dest)
			if err != nil {
				return err
			}
		case arg == "-var-file":
			// If the flag is just -var-file, we need to use the next arg as the value
			// and skip this one
			if i+1 < len(args) {
				err := ParseVarsFile(args[i+1], dest)
				if err != nil {
					return err
				}
			} else {
				continue
			}
		default:
			continue
		}

	}

	return nil
}

// Parses the value of a -var flag. The value should already be extracted here
// i.e. the text after the = sign, or after the space if the = sign isn't used,
// so you should be passing in "foo=var" or "[1,2,3]" etc.
//
// Terraform allows a user to specify string values without quotes,
// which isn't valid HCL, but everything else needs to be valid HCL. For
// example you can set a string like this:
//
//	-var="foo=bar"
//
// But this isn't valid HCL since the string isn't quoted. However if
// you want to set a list, map etc, you need to use valid HCL syntax.
// e.g.
//
//	-var="foo=[1,2,3]"
//
// In order to handle this we're going to try to parse as HCL, then
// fall back to basic string parsing if that doesn't work, which seems
// to be how the Terraform works
func ParseFlagValue(value string, dest *hcl.EvalContext) error {
	err := func() error {
		// Parse argument as HCL
		parser := hclparse.NewParser()
		parsedFile, diag := parser.ParseHCL([]byte(value), "")
		if diag.HasErrors() {
			return fmt.Errorf("error parsing terraform vars file: %w", diag)
		}

		// Decode the body
		var vars map[string]cty.Value
		diag = gohcl.DecodeBody(parsedFile.Body, nil, &vars)
		if diag.HasErrors() {
			return fmt.Errorf("error decoding terraform vars file: %w", diag)
		}

		// Merge the vars into the eval context
		for k, v := range vars {
			dest.Variables[k] = v
		}

		return nil
	}()

	if err != nil {
		// Fall back to string parsing
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable argument: %s", value)
		}

		dest.Variables[parts[0]] = cty.StringVal(parts[1])
	}

	return nil
}

type ProviderResult struct {
	Provider *AWSProvider
	Error    error
	FilePath string
}

// Parses AWS provider config from all terraform files in the given directory,
// without recursion as we don't yet support providers in submodules. Returns a
// list of AWS providers and a list of files that were parsed. This will return
// an error only if there was an error loading the files. ProviderResults will
// be returned for:
//
// * Files that could not be parsed at all (just an error)
// * Files that contained an AWS provider that we couldn't fully evaluate (with an error)
// * Files that contained an AWS provider that we could fully evaluate (with no error)
func ParseAWSProviders(terraformDir string, evalContext *hcl.EvalContext) ([]ProviderResult, error) {
	files, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	if err != nil {
		return nil, err
	}

	parser := hclparse.NewParser()
	results := make([]ProviderResult, 0)

	// Iterate over the files
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			results = append(results, ProviderResult{
				Error:    fmt.Errorf("error reading terraform file: (%v) %w", file, err),
				FilePath: file,
			})
			continue
		}

		// Parse the HCL file
		parsedFile, diag := parser.ParseHCL(b, file)
		if diag.HasErrors() {
			results = append(results, ProviderResult{
				Error:    fmt.Errorf("error parsing terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		providerFile := ProviderFile{}
		diag = gohcl.DecodeBody(parsedFile.Body, evalContext, &providerFile)
		if diag.HasErrors() {
			results = append(results, ProviderResult{
				Error:    fmt.Errorf("error decoding terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		for _, provider := range providerFile.Providers {
			if provider.Name == "aws" {
				results = append(results, ProviderResult{
					Provider: &provider, //nolint:all // this is not relevant for go1.22 and later
					FilePath: file,
				})
			}
		}
	}

	return results, nil
}

// ConfigFromProvider creates an aws.Config from an AWSProvider that uses the
// provided HTTP client. This client will be modified with proxy settings if
// they are present in the provider.
func ConfigFromProvider(ctx context.Context, provider AWSProvider) (aws.Config, error) {
	var options []func(*config.LoadOptions) error

	if provider.AccessKey != "" {
		options = append(options, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     provider.AccessKey,
				SecretAccessKey: provider.SecretKey,
				SessionToken:    provider.Token,
			},
		}))
	}

	if provider.Region != "" {
		options = append(options, config.WithRegion(provider.Region))
	}

	if provider.CustomCABundle != "" {
		bundlePath := os.ExpandEnv(provider.CustomCABundle)
		bundlePath, err := homedir.Expand(bundlePath)
		if err != nil {
			return aws.Config{}, fmt.Errorf("expanding custom CA bundle path: %w", err)
		}

		bundle, err := os.ReadFile(bundlePath)
		if err != nil {
			return aws.Config{}, fmt.Errorf("reading custom CA bundle: %w", err)
		}

		options = append(options, config.WithCustomCABundle(bytes.NewReader(bundle)))
	}

	if provider.EC2MetadataServiceEndpoint != "" {
		options = append(options, config.WithEC2IMDSEndpoint(provider.EC2MetadataServiceEndpoint))
	}

	if provider.EC2MetadataServiceEndpointMode != "" {
		var mode imds.EndpointModeState

		switch {
		case len(provider.EC2MetadataServiceEndpointMode) == 0:
			mode = imds.EndpointModeStateUnset
		case strings.EqualFold(provider.EC2MetadataServiceEndpointMode, "IPv6"):
			mode = imds.EndpointModeStateIPv4
		case strings.EqualFold(provider.EC2MetadataServiceEndpointMode, "IPv4"):
			mode = imds.EndpointModeStateIPv6
		default:
			return aws.Config{}, fmt.Errorf("unknown EC2 IMDS endpoint mode, must be either IPv6 or IPv4")
		}

		options = append(options, config.WithEC2IMDSEndpointMode(mode))
	}

	if provider.SkipMetadataAPICheck {
		options = append(options, config.WithEC2IMDSClientEnableState(imds.ClientDisabled))
	}

	proxyConfig := httpproxy.FromEnvironment()

	if provider.HTTPProxy != "" {
		proxyConfig.HTTPProxy = provider.HTTPProxy
	}

	if provider.HTTPSProxy != "" {
		proxyConfig.HTTPSProxy = provider.HTTPSProxy
	}

	if provider.NoProxy != "" {
		proxyConfig.NoProxy = provider.NoProxy
	}

	// Always append the HTTP client that is configured with all our required
	// proxy settings
	// TODO: Can we inherit a transport here for things like OTEL?
	httpClient := awshttp.NewBuildableClient()
	httpClient.WithTransportOptions(func(t *http.Transport) {
		t.Proxy = func(r *http.Request) (*url.URL, error) {
			return proxyConfig.ProxyFunc()(r.URL)
		}
	})
	options = append(options, config.WithHTTPClient(httpClient))

	if provider.MaxRetries != 0 {
		options = append(options, config.WithRetryMaxAttempts(provider.MaxRetries))
	}

	if provider.Profile != "" {
		options = append(options, config.WithSharedConfigProfile(provider.Profile))
	}

	if provider.RetryMode != "" {
		switch {
		case strings.EqualFold(provider.RetryMode, "standard"):
			options = append(options, config.WithRetryMode(aws.RetryModeStandard))
		case strings.EqualFold(provider.RetryMode, "adaptive"):
			options = append(options, config.WithRetryMode(aws.RetryModeAdaptive))
		default:
			return aws.Config{}, fmt.Errorf("unknown retry mode: %s. Must be 'standard' or 'adaptive'", provider.RetryMode)
		}
	}

	if len(provider.SharedConfigFiles) != 0 {
		options = append(options, config.WithSharedConfigFiles(provider.SharedConfigFiles))
	}

	if len(provider.SharedCredentialsFiles) != 0 {
		options = append(options, config.WithSharedCredentialsFiles(provider.SharedCredentialsFiles))
	}

	if provider.UseDualStackEndpoint {
		options = append(options, config.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled))
	}

	if provider.UseFIPSEndpoint {
		options = append(options, config.WithUseFIPSEndpoint(aws.FIPSEndpointStateEnabled))
	}

	return config.LoadDefaultConfig(ctx, options...)
}
