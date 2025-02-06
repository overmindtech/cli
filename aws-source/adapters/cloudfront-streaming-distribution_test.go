package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (t TestCloudFrontClient) GetStreamingDistribution(ctx context.Context, params *cloudfront.GetStreamingDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetStreamingDistributionOutput, error) {
	return &cloudfront.GetStreamingDistributionOutput{
		ETag: adapterhelpers.PtrString("E2QWRUHAPOMQZL"),
		StreamingDistribution: &types.StreamingDistribution{
			ARN:              adapterhelpers.PtrString("arn:aws:cloudfront::123456789012:streaming-distribution/EDFDVBD632BHDS5"),
			DomainName:       adapterhelpers.PtrString("d111111abcdef8.cloudfront.net"), // link
			Id:               adapterhelpers.PtrString("EDFDVBD632BHDS5"),
			Status:           adapterhelpers.PtrString("Deployed"), // health
			LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
			ActiveTrustedSigners: &types.ActiveTrustedSigners{
				Enabled:  adapterhelpers.PtrBool(true),
				Quantity: adapterhelpers.PtrInt32(1),
				Items: []types.Signer{
					{
						AwsAccountNumber: adapterhelpers.PtrString("123456789012"),
						KeyPairIds: &types.KeyPairIds{
							Quantity: adapterhelpers.PtrInt32(1),
							Items: []string{
								"APKAJDGKZRVEXAMPLE",
							},
						},
					},
				},
			},
			StreamingDistributionConfig: &types.StreamingDistributionConfig{
				CallerReference: adapterhelpers.PtrString("test"),
				Comment:         adapterhelpers.PtrString("test"),
				Enabled:         adapterhelpers.PtrBool(true),
				S3Origin: &types.S3Origin{
					DomainName:           adapterhelpers.PtrString("myawsbucket.s3.amazonaws.com"),                     // link
					OriginAccessIdentity: adapterhelpers.PtrString("origin-access-identity/cloudfront/E127EXAMPLE51Z"), // link
				},
				TrustedSigners: &types.TrustedSigners{
					Enabled:  adapterhelpers.PtrBool(true),
					Quantity: adapterhelpers.PtrInt32(1),
					Items: []string{
						"self",
					},
				},
				Aliases: &types.Aliases{
					Quantity: adapterhelpers.PtrInt32(1),
					Items: []string{
						"example.com", // link
					},
				},
				Logging: &types.StreamingLoggingConfig{
					Bucket:  adapterhelpers.PtrString("myawslogbucket.s3.amazonaws.com"), // link
					Enabled: adapterhelpers.PtrBool(true),
					Prefix:  adapterhelpers.PtrString("myprefix"),
				},
				PriceClass: types.PriceClassPriceClassAll,
			},
		},
	}, nil
}

func (t TestCloudFrontClient) ListStreamingDistributions(ctx context.Context, params *cloudfront.ListStreamingDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListStreamingDistributionsOutput, error) {
	return &cloudfront.ListStreamingDistributionsOutput{
		StreamingDistributionList: &types.StreamingDistributionList{
			IsTruncated: adapterhelpers.PtrBool(false),
			Items: []types.StreamingDistributionSummary{
				{
					Id: adapterhelpers.PtrString("test-id"),
				},
			},
		},
	}, nil
}

func TestStreamingDistributionGetFunc(t *testing.T) {
	item, err := streamingDistributionGetFunc(context.Background(), TestCloudFrontClient{}, "foo", &cloudfront.GetStreamingDistributionInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Errorf("expected health to be HEALTH_OK, got %s", item.GetHealth())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "d111111abcdef8.cloudfront.net",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}

func TestNewCloudfrontStreamingDistributionAdapter(t *testing.T) {
	config, account, _ := adapterhelpers.GetAutoConfig(t)
	client := cloudfront.NewFromConfig(config)

	adapter := NewCloudfrontStreamingDistributionAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
