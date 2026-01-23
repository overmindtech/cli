package shared

import (
	"reflect"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestAWSLinkByARN(t *testing.T) {
	type args struct {
		awsItem          string
		blastPropagation *sdp.BlastPropagation
	}

	tests := []struct {
		name string
		arn  string
		args args
		want *sdp.LinkedItemQuery
	}{
		{
			name: "Link by ARN for AWS IAM Role - global scope",
			arn:  "arn:aws:iam::123456789012:role/MyRole",
			args: args{
				awsItem: "iam-role",
				blastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:iam::123456789012:role/MyRole",
					Scope:  "123456789012",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		},
		{
			name: "Link by ARN for AWS KMS Key - region scope",
			arn:  "arn:aws:kms:us-west-2:123456789012:key/abcd1234-56ef-78gh-90ij-klmnopqrstuv",
			args: args{
				awsItem: "kms-key",
				blastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "kms-key",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:kms:us-west-2:123456789012:key/abcd1234-56ef-78gh-90ij-klmnopqrstuv",
					Scope:  "123456789012.us-west-2", // Region scope
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		},
		{
			name: "Malformed ARN",
			arn:  "invalid-arn",
			args: args{
				awsItem:          "iam-role",
				blastPropagation: &sdp.BlastPropagation{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFunc := AWSLinkByARN(tt.args.awsItem)
			gotLIQ := gotFunc("", "", tt.arn, tt.args.blastPropagation)
			if !reflect.DeepEqual(gotLIQ, tt.want) {
				t.Errorf("AWSLinkByARN() = %v, want %v", gotLIQ, tt.want)
			}
		})
	}
}

func TestForwardingRuleTargetLinker(t *testing.T) {
	projectID := "test-project"
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: true,
	}

	tests := []struct {
		name      string
		targetURI string
		want      *sdp.LinkedItemQuery
	}{
		// Global Target HTTP Proxy tests
		{
			name:      "Global Target HTTP Proxy - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/global/targetHttpProxies/my-http-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-http-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Global Target HTTP Proxy - resource name format",
			targetURI: "projects/test-project/global/targetHttpProxies/my-http-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-http-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Global Target HTTP Proxy - compute.googleapis.com URL",
			targetURI: "https://compute.googleapis.com/compute/v1/projects/test-project/global/targetHttpProxies/my-http-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-http-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Global Target HTTPS Proxy tests
		{
			name:      "Global Target HTTPS Proxy - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/global/targetHttpsProxies/my-https-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpsProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-https-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Global Target HTTPS Proxy - resource name format",
			targetURI: "projects/test-project/global/targetHttpsProxies/my-https-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpsProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-https-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Global Target TCP Proxy tests
		{
			name:      "Global Target TCP Proxy - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/global/targetTcpProxies/my-tcp-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetTcpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-tcp-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Global Target TCP Proxy - resource name format",
			targetURI: "projects/test-project/global/targetTcpProxies/my-tcp-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetTcpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-tcp-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Global Target SSL Proxy tests
		{
			name:      "Global Target SSL Proxy - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/global/targetSslProxies/my-ssl-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetSslProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-ssl-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Global Target SSL Proxy - resource name format",
			targetURI: "projects/test-project/global/targetSslProxies/my-ssl-proxy",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetSslProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-ssl-proxy",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Regional Target Pool tests
		{
			name:      "Regional Target Pool - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/targetPools/my-target-pool",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-target-pool",
					Scope:  "test-project.us-central1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Regional Target Pool - resource name format",
			targetURI: "projects/test-project/regions/us-central1/targetPools/my-target-pool",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-target-pool",
					Scope:  "test-project.us-central1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Regional Target VPN Gateway tests
		{
			name:      "Regional Target VPN Gateway - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-west1/targetVpnGateways/my-vpn-gateway",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetVpnGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-vpn-gateway",
					Scope:  "test-project.us-west1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Regional Target VPN Gateway - resource name format",
			targetURI: "projects/test-project/regions/us-west1/targetVpnGateways/my-vpn-gateway",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetVpnGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-vpn-gateway",
					Scope:  "test-project.us-west1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Zonal Target Instance tests
		{
			name:      "Zonal Target Instance - full HTTPS URL",
			targetURI: "https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a/targetInstances/my-target-instance",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetInstance.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-target-instance",
					Scope:  "test-project.us-central1-a",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "Zonal Target Instance - resource name format",
			targetURI: "projects/test-project/zones/us-central1-a/targetInstances/my-target-instance",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetInstance.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-target-instance",
					Scope:  "test-project.us-central1-a",
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Edge cases
		{
			name:      "Empty target URI",
			targetURI: "",
			want:      nil,
		},
		{
			name:      "Unknown target type",
			targetURI: "projects/test-project/global/unknownResources/unknown",
			want:      nil,
		},
		{
			name:      "Malformed URI - no resource name (trailing slash)",
			targetURI: "projects/test-project/global/targetHttpProxies/",
			// LastPathComponent returns "targetHttpProxies" (the resource type) when URI ends with slash
			// This results in a link being created but with incorrect query value
			// TODO: This might need to be fixed to return nil for malformed URIs
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeTargetHttpProxy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "targetHttpProxies", // LastPathComponent returns this from trailing slash
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:      "URI without project path context",
			targetURI: "targetHttpProxies/my-proxy",
			// The function expects "/targetHttpProxies/" with slashes on both sides,
			// so this format won't match and returns nil
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ForwardingRuleTargetLinker(projectID, "", tt.targetURI, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ForwardingRuleTargetLinker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetworkDNSLinker(t *testing.T) {
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: true,
	}

	tests := []struct {
		name  string
		query string
		want  *sdp.LinkedItemQuery
	}{
		{
			name:  "Simple DNS name",
			query: "example.com",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "example.com",
					Scope:  "global",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:  "DNS name with subdomain",
			query: "api.example.com",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "api.example.com",
					Scope:  "global",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:  "Empty query",
			query: "",
			want:  nil,
		},
	}

	linkerFunc := ManualAdapterLinksByAssetType[stdlib.NetworkDNS]
	if linkerFunc == nil {
		t.Fatal("NetworkDNS linker function not found in ManualAdapterLinksByAssetType")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linkerFunc("", "", tt.query, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NetworkDNSLinker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMSKClusterLinkByARN(t *testing.T) {
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: false,
	}

	tests := []struct {
		name string
		arn  string
		want *sdp.LinkedItemQuery
	}{
		{
			name: "MSK Cluster ARN with region",
			arn:  "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abcd1234-abcd-cafe-abab-9876543210ab-4",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "msk-cluster",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abcd1234-abcd-cafe-abab-9876543210ab-4",
					Scope:  "123456789012.us-east-1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name: "MSK Cluster ARN with different region",
			arn:  "arn:aws:kafka:us-west-2:987654321098:cluster/prod-cluster/efgh5678-efgh-cafe-cdcd-1234567890ab-5",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "msk-cluster",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:kafka:us-west-2:987654321098:cluster/prod-cluster/efgh5678-efgh-cafe-cdcd-1234567890ab-5",
					Scope:  "987654321098.us-west-2",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name: "Malformed ARN",
			arn:  "invalid-arn",
			want: nil,
		},
		{
			name: "Empty ARN",
			arn:  "",
			want: nil,
		},
	}

	linkerFunc := ManualAdapterLinksByAssetType[aws.MSKCluster]
	if linkerFunc == nil {
		t.Fatal("MSKCluster linker function not found in ManualAdapterLinksByAssetType")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linkerFunc("", "", tt.arn, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MSKClusterLinkByARN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthCheckLinker(t *testing.T) {
	projectID := "test-project"
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: false,
	}

	tests := []struct {
		name           string
		healthCheckURI string
		want           *sdp.LinkedItemQuery
	}{
		// Global Health Check tests
		{
			name:           "Global Health Check - full HTTPS URL",
			healthCheckURI: "https://compute.googleapis.com/compute/v1/projects/test-project/global/healthChecks/my-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-health-check",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:           "Global Health Check - resource name format",
			healthCheckURI: "projects/test-project/global/healthChecks/my-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-health-check",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:           "Global Health Check - www.googleapis.com URL",
			healthCheckURI: "https://www.googleapis.com/compute/v1/projects/test-project/global/healthChecks/my-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-health-check",
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Regional Health Check tests
		{
			name:           "Regional Health Check - full HTTPS URL",
			healthCheckURI: "https://compute.googleapis.com/compute/v1/projects/test-project/regions/us-central1/healthChecks/my-regional-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-regional-health-check",
					Scope:  "test-project.us-central1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:           "Regional Health Check - resource name format",
			healthCheckURI: "projects/test-project/regions/us-west1/healthChecks/my-regional-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-regional-health-check",
					Scope:  "test-project.us-west1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:           "Regional Health Check - different region",
			healthCheckURI: "https://www.googleapis.com/compute/v1/projects/test-project/regions/europe-west1/healthChecks/eu-health-check",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "eu-health-check",
					Scope:  "test-project.europe-west1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		// Edge cases
		{
			name:           "Empty health check URI",
			healthCheckURI: "",
			want:           nil,
		},
		{
			name:           "Not a health check URL",
			healthCheckURI: "projects/test-project/global/backendServices/my-backend-service",
			want:           nil,
		},
		{
			name:           "Malformed URI - no resource name",
			healthCheckURI: "projects/test-project/global/healthChecks/",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeHealthCheck.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "healthChecks", // LastPathComponent returns this from trailing slash
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HealthCheckLinker(projectID, "", tt.healthCheckURI, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HealthCheckLinker() = %v, want %v", got, tt.want)
			}
		})
	}
}
