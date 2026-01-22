package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

func TestResponseHeadersPolicyItemMapper(t *testing.T) {
	x := types.ResponseHeadersPolicy{
		Id:               PtrString("test"),
		LastModifiedTime: PtrTime(time.Now()),
		ResponseHeadersPolicyConfig: &types.ResponseHeadersPolicyConfig{
			Name:    PtrString("example-policy"),
			Comment: PtrString("example comment"),
			CorsConfig: &types.ResponseHeadersPolicyCorsConfig{
				AccessControlAllowCredentials: PtrBool(true),
				AccessControlAllowHeaders: &types.ResponseHeadersPolicyAccessControlAllowHeaders{
					Items:    []string{"X-Customer-Header"},
					Quantity: PtrInt32(1),
				},
			},
			CustomHeadersConfig: &types.ResponseHeadersPolicyCustomHeadersConfig{
				Quantity: PtrInt32(1),
				Items: []types.ResponseHeadersPolicyCustomHeader{
					{
						Header:   PtrString("X-Customer-Header"),
						Override: PtrBool(true),
						Value:    PtrString("test"),
					},
				},
			},
			RemoveHeadersConfig: &types.ResponseHeadersPolicyRemoveHeadersConfig{
				Quantity: PtrInt32(1),
				Items: []types.ResponseHeadersPolicyRemoveHeader{
					{
						Header: PtrString("X-Private-Header"),
					},
				},
			},
			SecurityHeadersConfig: &types.ResponseHeadersPolicySecurityHeadersConfig{
				ContentSecurityPolicy: &types.ResponseHeadersPolicyContentSecurityPolicy{
					ContentSecurityPolicy: PtrString("default-src 'none';"),
					Override:              PtrBool(true),
				},
				ContentTypeOptions: &types.ResponseHeadersPolicyContentTypeOptions{
					Override: PtrBool(true),
				},
				FrameOptions: &types.ResponseHeadersPolicyFrameOptions{
					FrameOption: types.FrameOptionsListDeny,
					Override:    PtrBool(true),
				},
				ReferrerPolicy: &types.ResponseHeadersPolicyReferrerPolicy{
					Override:       PtrBool(true),
					ReferrerPolicy: types.ReferrerPolicyListNoReferrer,
				},
				StrictTransportSecurity: &types.ResponseHeadersPolicyStrictTransportSecurity{
					AccessControlMaxAgeSec: PtrInt32(86400),
					Override:               PtrBool(true),
					IncludeSubdomains:      PtrBool(true),
					Preload:                PtrBool(true),
				},
				XSSProtection: &types.ResponseHeadersPolicyXSSProtection{
					Override:   PtrBool(true),
					Protection: PtrBool(true),
					ModeBlock:  PtrBool(true),
					ReportUri:  PtrString("https://example.com/report"),
				},
			},
			ServerTimingHeadersConfig: &types.ResponseHeadersPolicyServerTimingHeadersConfig{
				Enabled:      PtrBool(true),
				SamplingRate: PtrFloat64(0.1),
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

	adapter := NewCloudfrontResponseHeadersPolicyAdapter(client, account, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
