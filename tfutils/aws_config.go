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

// Loads the eval context from the following locations:
//
//   - `-var-file=FILENAME` files: These should be passed as file paths
//   - `-var 'NAME=VALUE'` arguments: These should be passed as a list of strings
//   - Environment Variables: These should be passed as a []strings (from `os.Environ()`),
//     variables beginning with TF_VAR_ will be used
func LoadEvalContext(varsFiles []string, args []string, env []string) (*hcl.EvalContext, error) {
	return nil, nil
}

// Parses AWS provider config from all terraform files in the given directory,
// recursing into subdirectories. Returns a list of AWS providers and a list of
// files that were parsed.
func ParseAWSProviders(terraformDir string, evalContext *hcl.EvalContext) ([]AWSProvider, []string, error) {
	files := make([]string, 0)

	// Get all files matching *.tf from everywhere under the directory
	err := filepath.Walk(terraformDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error searching for terraform files: %w", err)
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".tf" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	parser := hclparse.NewParser()
	awsProviders := make([]AWSProvider, 0)

	// TODO: We need to also make sure we have all the variables and inputs set
	// up so that dynamic values can be used. These could come from -vars-file
	// or the Environment. It's also possible to have a provider in a module
	// that sets parameters based on the input variables of the module
	//
	// * [ ] Parse tfvars files and arguments
	// * [ ] Parse environment variables

	// Iterate over the files
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, files, fmt.Errorf("error reading terraform file: (%v) %w", file, err)
		}

		// Parse the HCL file
		parsedFile, diag := parser.ParseHCL(b, file)
		if diag.HasErrors() {
			return nil, files, fmt.Errorf("error parsing terraform file: (%v) %w", file, diag)
		}

		providerFile := ProviderFile{}
		diag = gohcl.DecodeBody(parsedFile.Body, evalContext, &providerFile)
		if diag.HasErrors() {
			return nil, files, fmt.Errorf("error decoding terraform file: (%v) %w", file, diag)
		}

		for _, provider := range providerFile.Providers {
			if provider.Name == "aws" {
				awsProviders = append(awsProviders, provider)
			}
		}
	}

	return awsProviders, files, nil
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
