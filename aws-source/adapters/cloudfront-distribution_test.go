package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func (t TestCloudFrontClient) GetDistribution(ctx context.Context, params *cloudfront.GetDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error) {
	return &cloudfront.GetDistributionOutput{
		Distribution: &types.Distribution{
			ARN:                           new("arn:aws:cloudfront::123456789012:distribution/test-id"),
			DomainName:                    new("d111111abcdef8.cloudfront.net"), // link
			Id:                            new("test-id"),
			InProgressInvalidationBatches: new(int32(1)),
			LastModifiedTime:              new(time.Now()),
			Status:                        new("Deployed"), // health: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/distribution-web-values-returned.html
			ActiveTrustedKeyGroups: &types.ActiveTrustedKeyGroups{
				Enabled:  new(true),
				Quantity: new(int32(1)),
				Items: []types.KGKeyPairIds{
					{
						KeyGroupId: new("key-group-1"), // link
						KeyPairIds: &types.KeyPairIds{
							Quantity: new(int32(1)),
							Items: []string{
								"123456789",
							},
						},
					},
				},
			},
			ActiveTrustedSigners: &types.ActiveTrustedSigners{
				Enabled:  new(true),
				Quantity: new(int32(1)),
				Items: []types.Signer{
					{
						AwsAccountNumber: new("123456789"),
						KeyPairIds: &types.KeyPairIds{
							Quantity: new(int32(1)),
							Items: []string{
								"123456789",
							},
						},
					},
				},
			},
			AliasICPRecordals: []types.AliasICPRecordal{
				{
					CNAME:             new("something.foo.bar.com"), // link
					ICPRecordalStatus: types.ICPRecordalStatusApproved,
				},
			},
			DistributionConfig: &types.DistributionConfig{
				CallerReference: new("test-caller-reference"),
				Comment:         new("test-comment"),
				Enabled:         new(true),
				Aliases: &types.Aliases{
					Quantity: new(int32(1)),
					Items: []string{
						"www.example.com", // link
					},
				},
				Staging:                      new(true),
				ContinuousDeploymentPolicyId: new("test-continuous-deployment-policy-id"), // link
				CacheBehaviors: &types.CacheBehaviors{
					Quantity: new(int32(1)),
					Items: []types.CacheBehavior{
						{
							PathPattern:          new("/foo"),
							TargetOriginId:       new("CustomOriginConfig"),
							ViewerProtocolPolicy: types.ViewerProtocolPolicyHttpsOnly,
							AllowedMethods: &types.AllowedMethods{
								Items: []types.Method{
									types.MethodGet,
								},
							},
							CachePolicyId:           new("test-cache-policy-id"), // link
							Compress:                new(true),
							DefaultTTL:              new(int64(1)),
							FieldLevelEncryptionId:  new("test-field-level-encryption-id"), // link
							MaxTTL:                  new(int64(1)),
							MinTTL:                  new(int64(1)),
							OriginRequestPolicyId:   new("test-origin-request-policy-id"),                                   // link
							RealtimeLogConfigArn:    new("arn:aws:logs:us-east-1:123456789012:realtime-log-config/test-id"), // link
							ResponseHeadersPolicyId: new("test-response-headers-policy-id"),                                 // link
							SmoothStreaming:         new(true),
							TrustedKeyGroups: &types.TrustedKeyGroups{
								Enabled:  new(true),
								Quantity: new(int32(1)),
								Items: []string{
									"key-group-1", // link
								},
							},
							TrustedSigners: &types.TrustedSigners{
								Enabled:  new(true),
								Quantity: new(int32(1)),
								Items: []string{
									"123456789",
								},
							},
							ForwardedValues: &types.ForwardedValues{
								Cookies: &types.CookiePreference{
									Forward: types.ItemSelectionWhitelist,
									WhitelistedNames: &types.CookieNames{
										Quantity: new(int32(1)),
										Items: []string{
											"cookie_123",
										},
									},
								},
								QueryString: new(true),
								Headers: &types.Headers{
									Quantity: new(int32(1)),
									Items: []string{
										"X-Customer-Header",
									},
								},
								QueryStringCacheKeys: &types.QueryStringCacheKeys{
									Quantity: new(int32(1)),
									Items: []string{
										"test-query-string-cache-key",
									},
								},
							},
							FunctionAssociations: &types.FunctionAssociations{
								Quantity: new(int32(1)),
								Items: []types.FunctionAssociation{
									{
										EventType:   types.EventTypeOriginRequest,
										FunctionARN: new("arn:aws:cloudfront::123412341234:function/1234"), // link
									},
								},
							},
							LambdaFunctionAssociations: &types.LambdaFunctionAssociations{
								Quantity: new(int32(1)),
								Items: []types.LambdaFunctionAssociation{
									{
										EventType:         types.EventTypeOriginResponse,
										LambdaFunctionARN: new("arn:aws:lambda:us-east-1:123456789012:function:test-function"), // link
										IncludeBody:       new(true),
									},
								},
							},
						},
					},
				},
				Origins: &types.Origins{
					Items: []types.Origin{
						{
							DomainName:         new("DOC-EXAMPLE-BUCKET.s3.us-west-2.amazonaws.com"), // link
							Id:                 new("CustomOriginConfig"),
							ConnectionAttempts: new(int32(3)),
							ConnectionTimeout:  new(int32(10)),
							CustomHeaders: &types.CustomHeaders{
								Quantity: new(int32(1)),
								Items: []types.OriginCustomHeader{
									{
										HeaderName:  new("test-header-name"),
										HeaderValue: new("test-header-value"),
									},
								},
							},
							CustomOriginConfig: &types.CustomOriginConfig{
								HTTPPort:               new(int32(80)),
								HTTPSPort:              new(int32(443)),
								OriginProtocolPolicy:   types.OriginProtocolPolicyMatchViewer,
								OriginKeepaliveTimeout: new(int32(5)),
								OriginReadTimeout:      new(int32(30)),
								OriginSslProtocols: &types.OriginSslProtocols{
									Items: types.SslProtocolSSLv3.Values(),
								},
							},
							OriginAccessControlId: new("test-origin-access-control-id"), // link
							OriginPath:            new("/foo"),
							OriginShield: &types.OriginShield{
								Enabled:            new(true),
								OriginShieldRegion: new("eu-west-1"),
							},
							S3OriginConfig: &types.S3OriginConfig{
								OriginAccessIdentity: new("test-origin-access-identity"), // link
							},
						},
					},
				},
				DefaultCacheBehavior: &types.DefaultCacheBehavior{
					TargetOriginId:          new("CustomOriginConfig"),
					ViewerProtocolPolicy:    types.ViewerProtocolPolicyHttpsOnly,
					CachePolicyId:           new("test-cache-policy-id"), // link
					Compress:                new(true),
					DefaultTTL:              new(int64(1)),
					FieldLevelEncryptionId:  new("test-field-level-encryption-id"), // link
					MaxTTL:                  new(int64(1)),
					MinTTL:                  new(int64(1)),
					OriginRequestPolicyId:   new("test-origin-request-policy-id"),                                   // link
					RealtimeLogConfigArn:    new("arn:aws:logs:us-east-1:123456789012:realtime-log-config/test-id"), // link
					ResponseHeadersPolicyId: new("test-response-headers-policy-id"),                                 // link
					SmoothStreaming:         new(true),
					ForwardedValues: &types.ForwardedValues{
						Cookies: &types.CookiePreference{
							Forward: types.ItemSelectionWhitelist,
							WhitelistedNames: &types.CookieNames{
								Quantity: new(int32(1)),
								Items: []string{
									"cooke_123",
								},
							},
						},
						QueryString: new(true),
						Headers: &types.Headers{
							Quantity: new(int32(1)),
							Items: []string{
								"X-Customer-Header",
							},
						},
						QueryStringCacheKeys: &types.QueryStringCacheKeys{
							Quantity: new(int32(1)),
							Items: []string{
								"test-query-string-cache-key",
							},
						},
					},
					FunctionAssociations: &types.FunctionAssociations{
						Quantity: new(int32(1)),
						Items: []types.FunctionAssociation{
							{
								EventType:   types.EventTypeViewerRequest,
								FunctionARN: new("arn:aws:cloudfront::123412341234:function/1234"), // link
							},
						},
					},
					LambdaFunctionAssociations: &types.LambdaFunctionAssociations{
						Quantity: new(int32(1)),
						Items: []types.LambdaFunctionAssociation{
							{
								EventType:         types.EventTypeOriginRequest,
								LambdaFunctionARN: new("arn:aws:lambda:us-east-1:123456789012:function:test-function"), // link
								IncludeBody:       new(true),
							},
						},
					},
					TrustedKeyGroups: &types.TrustedKeyGroups{
						Enabled:  new(true),
						Quantity: new(int32(1)),
						Items: []string{
							"key-group-1", // link
						},
					},
					TrustedSigners: &types.TrustedSigners{
						Enabled:  new(true),
						Quantity: new(int32(1)),
						Items: []string{
							"123456789",
						},
					},
					AllowedMethods: &types.AllowedMethods{
						Items: []types.Method{
							types.MethodGet,
						},
						Quantity: new(int32(1)),
						CachedMethods: &types.CachedMethods{
							Items: []types.Method{
								types.MethodGet,
							},
						},
					},
				},
				CustomErrorResponses: &types.CustomErrorResponses{
					Quantity: new(int32(1)),
					Items: []types.CustomErrorResponse{
						{
							ErrorCode:          new(int32(404)),
							ErrorCachingMinTTL: new(int64(1)),
							ResponseCode:       new("200"),
							ResponsePagePath:   new("/foo"),
						},
					},
				},
				DefaultRootObject: new("index.html"),
				HttpVersion:       types.HttpVersionHttp11,
				IsIPV6Enabled:     new(true),
				Logging: &types.LoggingConfig{
					Bucket:         new("aws-cf-access-logs.s3.amazonaws.com"), // link
					Enabled:        new(true),
					IncludeCookies: new(true),
					Prefix:         new("test-prefix"),
				},
				OriginGroups: &types.OriginGroups{
					Quantity: new(int32(1)),
					Items: []types.OriginGroup{
						{
							FailoverCriteria: &types.OriginGroupFailoverCriteria{
								StatusCodes: &types.StatusCodes{
									Items: []int32{
										404,
									},
									Quantity: new(int32(1)),
								},
							},
							Id: new("test-id"),
							Members: &types.OriginGroupMembers{
								Quantity: new(int32(1)),
								Items: []types.OriginGroupMember{
									{
										OriginId: new("CustomOriginConfig"),
									},
								},
							},
						},
					},
				},
				PriceClass: types.PriceClassPriceClass200,
				Restrictions: &types.Restrictions{
					GeoRestriction: &types.GeoRestriction{
						Quantity:        new(int32(1)),
						RestrictionType: types.GeoRestrictionTypeWhitelist,
						Items: []string{
							"US",
						},
					},
				},
				ViewerCertificate: &types.ViewerCertificate{
					ACMCertificateArn:            new("arn:aws:acm:us-east-1:123456789012:certificate/test-id"), // link
					Certificate:                  new("test-certificate"),
					CertificateSource:            types.CertificateSourceAcm,
					CloudFrontDefaultCertificate: new(true),
					IAMCertificateId:             new("test-iam-certificate-id"), // link
					MinimumProtocolVersion:       types.MinimumProtocolVersion(types.SslProtocolSSLv3),
					SSLSupportMethod:             types.SSLSupportMethodSniOnly,
				},
				// Note this can also be in the format: 473e64fd-f30b-4765-81a0-62ad96dd167a for WAF Classic
				WebACLId: new("arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a"), // link
			},
		},
	}, nil
}

func (t TestCloudFrontClient) ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error) {
	return &cloudfront.ListDistributionsOutput{
		DistributionList: &types.DistributionList{
			IsTruncated: new(false),
			Items: []types.DistributionSummary{
				{
					Id: new("test-id"),
				},
			},
		},
	}, nil
}

func TestDistributionGetFunc(t *testing.T) {
	scope := "123456789012"
	item, err := distributionGetFunc(context.Background(), TestCloudFrontClient{}, scope, &cloudfront.GetDistributionInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Errorf("expected health to be HEALTH_OK, got %s", item.GetHealth())
	}

	tests := QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "d111111abcdef8.cloudfront.net",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "cloudfront-key-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "key-group-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "something.foo.bar.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "cloudfront-continuous-deployment-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-continuous-deployment-policy-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-cache-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-cache-policy-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-field-level-encryption",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-field-level-encryption-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-origin-request-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-origin-request-policy-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-realtime-log-config",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:logs:us-east-1:123456789012:realtime-log-config/test-id",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "cloudfront-response-headers-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-response-headers-policy-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-key-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "key-group-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-function",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:cloudfront::123412341234:function/1234",
			ExpectedScope:  "123412341234",
		},
		{
			ExpectedType:   "lambda-function",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:lambda:us-east-1:123456789012:function:test-function",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "DOC-EXAMPLE-BUCKET.s3.us-west-2.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "cloudfront-origin-access-control",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-origin-access-control-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "cloudfront-cloud-front-origin-access-identity",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-origin-access-identity",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "aws-cf-access-logs.s3.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "acm-certificate",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:acm:us-east-1:123456789012:certificate/test-id",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "iam-server-certificate",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-iam-certificate-id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "wafv2-web-acl",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "DOC-EXAMPLE-BUCKET",
			ExpectedScope:  "123456789012",
		},
	}

	tests.Execute(t, item)
}

func TestNewCloudfrontDistributionAdapter(t *testing.T) {
	config, account, _ := GetAutoConfig(t)
	client := cloudfront.NewFromConfig(config)

	adapter := NewCloudfrontDistributionAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
