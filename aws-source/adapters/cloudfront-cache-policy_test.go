package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

var testCachePolicy = &types.CachePolicy{
	Id:               adapterhelpers.PtrString("test-id"),
	LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
	CachePolicyConfig: &types.CachePolicyConfig{
		MinTTL:     adapterhelpers.PtrInt64(1),
		Name:       adapterhelpers.PtrString("test-name"),
		Comment:    adapterhelpers.PtrString("test-comment"),
		DefaultTTL: adapterhelpers.PtrInt64(1),
		MaxTTL:     adapterhelpers.PtrInt64(1),
		ParametersInCacheKeyAndForwardedToOrigin: &types.ParametersInCacheKeyAndForwardedToOrigin{
			CookiesConfig: &types.CachePolicyCookiesConfig{
				CookieBehavior: types.CachePolicyCookieBehaviorAll,
				Cookies: &types.CookieNames{
					Quantity: adapterhelpers.PtrInt32(1),
					Items: []string{
						"test-cookie",
					},
				},
			},
			EnableAcceptEncodingGzip: adapterhelpers.PtrBool(true),
			HeadersConfig: &types.CachePolicyHeadersConfig{
				HeaderBehavior: types.CachePolicyHeaderBehaviorWhitelist,
				Headers: &types.Headers{
					Quantity: adapterhelpers.PtrInt32(1),
					Items: []string{
						"test-header",
					},
				},
			},
			QueryStringsConfig: &types.CachePolicyQueryStringsConfig{
				QueryStringBehavior: types.CachePolicyQueryStringBehaviorWhitelist,
				QueryStrings: &types.QueryStringNames{
					Quantity: adapterhelpers.PtrInt32(1),
					Items: []string{
						"test-query-string",
					},
				},
			},
			EnableAcceptEncodingBrotli: adapterhelpers.PtrBool(true),
		},
	},
}

func (t TestCloudFrontClient) ListCachePolicies(ctx context.Context, params *cloudfront.ListCachePoliciesInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListCachePoliciesOutput, error) {
	return &cloudfront.ListCachePoliciesOutput{
		CachePolicyList: &types.CachePolicyList{
			Items: []types.CachePolicySummary{
				{
					Type:        types.CachePolicyTypeManaged,
					CachePolicy: testCachePolicy,
				},
			},
		},
	}, nil
}

func (t TestCloudFrontClient) GetCachePolicy(ctx context.Context, params *cloudfront.GetCachePolicyInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetCachePolicyOutput, error) {
	return &cloudfront.GetCachePolicyOutput{
		CachePolicy: testCachePolicy,
	}, nil
}

func TestCachePolicyListFunc(t *testing.T) {
	policies, err := cachePolicyListFunc(context.Background(), TestCloudFrontClient{}, "aws")

	if err != nil {
		t.Fatal(err)
	}

	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
}

func TestNewCloudfrontCachePolicyAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontCachePolicyAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
