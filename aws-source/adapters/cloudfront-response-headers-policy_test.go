package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestResponseHeadersPolicyItemMapper(t *testing.T) {
	x := types.ResponseHeadersPolicy{
		Id:               new("test"),
		LastModifiedTime: new(time.Now()),
		ResponseHeadersPolicyConfig: &types.ResponseHeadersPolicyConfig{
			Name:    new("example-policy"),
			Comment: new("example comment"),
			CorsConfig: &types.ResponseHeadersPolicyCorsConfig{
				AccessControlAllowCredentials: new(true),
				AccessControlAllowHeaders: &types.ResponseHeadersPolicyAccessControlAllowHeaders{
					Items:    []string{"X-Customer-Header"},
					Quantity: new(int32(1)),
				},
			},
			CustomHeadersConfig: &types.ResponseHeadersPolicyCustomHeadersConfig{
				Quantity: new(int32(1)),
				Items: []types.ResponseHeadersPolicyCustomHeader{
					{
						Header:   new("X-Customer-Header"),
						Override: new(true),
						Value:    new("test"),
					},
				},
			},
			RemoveHeadersConfig: &types.ResponseHeadersPolicyRemoveHeadersConfig{
				Quantity: new(int32(1)),
				Items: []types.ResponseHeadersPolicyRemoveHeader{
					{
						Header: new("X-Private-Header"),
					},
				},
			},
			SecurityHeadersConfig: &types.ResponseHeadersPolicySecurityHeadersConfig{
				ContentSecurityPolicy: &types.ResponseHeadersPolicyContentSecurityPolicy{
					ContentSecurityPolicy: new("default-src 'none';"),
					Override:              new(true),
				},
				ContentTypeOptions: &types.ResponseHeadersPolicyContentTypeOptions{
					Override: new(true),
				},
				FrameOptions: &types.ResponseHeadersPolicyFrameOptions{
					FrameOption: types.FrameOptionsListDeny,
					Override:    new(true),
				},
				ReferrerPolicy: &types.ResponseHeadersPolicyReferrerPolicy{
					Override:       new(true),
					ReferrerPolicy: types.ReferrerPolicyListNoReferrer,
				},
				StrictTransportSecurity: &types.ResponseHeadersPolicyStrictTransportSecurity{
					AccessControlMaxAgeSec: new(int32(86400)),
					Override:               new(true),
					IncludeSubdomains:      new(true),
					Preload:                new(true),
				},
				XSSProtection: &types.ResponseHeadersPolicyXSSProtection{
					Override:   new(true),
					Protection: new(true),
					ModeBlock:  new(true),
					ReportUri:  new("https://example.com/report"),
				},
			},
			ServerTimingHeadersConfig: &types.ResponseHeadersPolicyServerTimingHeadersConfig{
				Enabled:      new(true),
				SamplingRate: new(0.1),
			},
		},
	}

	item, err := ResponseHeadersPolicyItemMapper("", "test", &x)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewCloudfrontResponseHeadersPolicyAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontResponseHeadersPolicyAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
