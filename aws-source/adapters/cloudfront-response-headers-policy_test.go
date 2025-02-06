package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestResponseHeadersPolicyItemMapper(t *testing.T) {
	x := types.ResponseHeadersPolicy{
		Id:               adapterhelpers.PtrString("test"),
		LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
		ResponseHeadersPolicyConfig: &types.ResponseHeadersPolicyConfig{
			Name:    adapterhelpers.PtrString("example-policy"),
			Comment: adapterhelpers.PtrString("example comment"),
			CorsConfig: &types.ResponseHeadersPolicyCorsConfig{
				AccessControlAllowCredentials: adapterhelpers.PtrBool(true),
				AccessControlAllowHeaders: &types.ResponseHeadersPolicyAccessControlAllowHeaders{
					Items:    []string{"X-Customer-Header"},
					Quantity: adapterhelpers.PtrInt32(1),
				},
			},
			CustomHeadersConfig: &types.ResponseHeadersPolicyCustomHeadersConfig{
				Quantity: adapterhelpers.PtrInt32(1),
				Items: []types.ResponseHeadersPolicyCustomHeader{
					{
						Header:   adapterhelpers.PtrString("X-Customer-Header"),
						Override: adapterhelpers.PtrBool(true),
						Value:    adapterhelpers.PtrString("test"),
					},
				},
			},
			RemoveHeadersConfig: &types.ResponseHeadersPolicyRemoveHeadersConfig{
				Quantity: adapterhelpers.PtrInt32(1),
				Items: []types.ResponseHeadersPolicyRemoveHeader{
					{
						Header: adapterhelpers.PtrString("X-Private-Header"),
					},
				},
			},
			SecurityHeadersConfig: &types.ResponseHeadersPolicySecurityHeadersConfig{
				ContentSecurityPolicy: &types.ResponseHeadersPolicyContentSecurityPolicy{
					ContentSecurityPolicy: adapterhelpers.PtrString("default-src 'none';"),
					Override:              adapterhelpers.PtrBool(true),
				},
				ContentTypeOptions: &types.ResponseHeadersPolicyContentTypeOptions{
					Override: adapterhelpers.PtrBool(true),
				},
				FrameOptions: &types.ResponseHeadersPolicyFrameOptions{
					FrameOption: types.FrameOptionsListDeny,
					Override:    adapterhelpers.PtrBool(true),
				},
				ReferrerPolicy: &types.ResponseHeadersPolicyReferrerPolicy{
					Override:       adapterhelpers.PtrBool(true),
					ReferrerPolicy: types.ReferrerPolicyListNoReferrer,
				},
				StrictTransportSecurity: &types.ResponseHeadersPolicyStrictTransportSecurity{
					AccessControlMaxAgeSec: adapterhelpers.PtrInt32(86400),
					Override:               adapterhelpers.PtrBool(true),
					IncludeSubdomains:      adapterhelpers.PtrBool(true),
					Preload:                adapterhelpers.PtrBool(true),
				},
				XSSProtection: &types.ResponseHeadersPolicyXSSProtection{
					Override:   adapterhelpers.PtrBool(true),
					Protection: adapterhelpers.PtrBool(true),
					ModeBlock:  adapterhelpers.PtrBool(true),
					ReportUri:  adapterhelpers.PtrString("https://example.com/report"),
				},
			},
			ServerTimingHeadersConfig: &types.ResponseHeadersPolicyServerTimingHeadersConfig{
				Enabled:      adapterhelpers.PtrBool(true),
				SamplingRate: adapterhelpers.PtrFloat64(0.1),
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

	adapter := NewCloudfrontResponseHeadersPolicyAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
