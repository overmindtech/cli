package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

var testCachePolicy = &types.CachePolicy{
	Id:               PtrString("test-id"),
	LastModifiedTime: PtrTime(time.Now()),
	CachePolicyConfig: &types.CachePolicyConfig{
		MinTTL:     PtrInt64(1),
		Name:       PtrString("test-name"),
		Comment:    PtrString("test-comment"),
		DefaultTTL: PtrInt64(1),
		MaxTTL:     PtrInt64(1),
		ParametersInCacheKeyAndForwardedToOrigin: &types.ParametersInCacheKeyAndForwardedToOrigin{
			CookiesConfig: &types.CachePolicyCookiesConfig{
				CookieBehavior: types.CachePolicyCookieBehaviorAll,
				Cookies: &types.CookieNames{
					Quantity: PtrInt32(1),
					Items: []string{
						"test-cookie",
					},
				},
			},
			EnableAcceptEncodingGzip: PtrBool(true),
			HeadersConfig: &types.CachePolicyHeadersConfig{
				HeaderBehavior: types.CachePolicyHeaderBehaviorWhitelist,
				Headers: &types.Headers{
					Quantity: PtrInt32(1),
					Items: []string{
						"test-header",
					},
				},
			},
			QueryStringsConfig: &types.CachePolicyQueryStringsConfig{
				QueryStringBehavior: types.CachePolicyQueryStringBehaviorWhitelist,
				QueryStrings: &types.QueryStringNames{
					Quantity: PtrInt32(1),
					Items: []string{
						"test-query-string",
					},
				},
			},
			EnableAcceptEncodingBrotli: PtrBool(true),
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

	adapter := NewCloudfrontCachePolicyAdapter(client, account, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
