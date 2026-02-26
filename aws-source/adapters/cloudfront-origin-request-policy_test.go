package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestOriginRequestPolicyItemMapper(t *testing.T) {
	x := types.OriginRequestPolicy{
		Id:               new("test"),
		LastModifiedTime: new(time.Now()),
		OriginRequestPolicyConfig: &types.OriginRequestPolicyConfig{
			Name:    new("example-policy"),
			Comment: new("example comment"),
			QueryStringsConfig: &types.OriginRequestPolicyQueryStringsConfig{
				QueryStringBehavior: types.OriginRequestPolicyQueryStringBehaviorAllExcept,
				QueryStrings: &types.QueryStringNames{
					Quantity: new(int32(1)),
					Items:    []string{"test"},
				},
			},
			CookiesConfig: &types.OriginRequestPolicyCookiesConfig{
				CookieBehavior: types.OriginRequestPolicyCookieBehaviorAll,
				Cookies: &types.CookieNames{
					Quantity: new(int32(1)),
					Items:    []string{"test"},
				},
			},
			HeadersConfig: &types.OriginRequestPolicyHeadersConfig{
				HeaderBehavior: types.OriginRequestPolicyHeaderBehaviorAllViewer,
				Headers: &types.Headers{
					Quantity: new(int32(1)),
					Items:    []string{"test"},
				},
			},
		},
	}

	item, err := originRequestPolicyItemMapper("", "test", &x)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewCloudfrontOriginRequestPolicyAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontOriginRequestPolicyAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
