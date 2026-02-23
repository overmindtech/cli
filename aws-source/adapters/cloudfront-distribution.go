package adapters

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

var s3DnsRegex = regexp.MustCompile(`([^\.]+)\.s3\.([^\.]+)\.amazonaws\.com`)

func distributionGetFunc(ctx context.Context, client CloudFrontClient, scope string, input *cloudfront.GetDistributionInput) (*sdp.Item, error) {
	out, err := client.GetDistribution(ctx, input)

	if err != nil {
		return nil, err
	}

	d := out.Distribution

	if d == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "distribution was nil",
		}
	}

	var tags map[string]string

	// get tags
	tagsOut, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
		Resource: d.ARN,
	})

	if err == nil {
		tags = cloudfrontTagsToMap(tagsOut.Tags)
	} else {
		tags = HandleTagsError(ctx, err)
	}

	attributes, err := ToAttributesWithExclude(d)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-distribution",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            tags,
	}

	if d.Status != nil {
		switch *d.Status {
		case "InProgress":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Deployed":
			item.Health = sdp.Health_HEALTH_OK.Enum()
		}
	}

	if d.DomainName != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *d.DomainName,
				Scope:  "global",
			},
		})
	}

	if d.ActiveTrustedKeyGroups != nil {
		for _, keyGroup := range d.ActiveTrustedKeyGroups.Items {
			if keyGroup.KeyGroupId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-key-group",
						Method: sdp.QueryMethod_GET,
						Query:  *keyGroup.KeyGroupId,
						Scope:  scope,
					},
				})
			}
		}
	}

	for _, record := range d.AliasICPRecordals {
		if record.CNAME != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *record.CNAME,
					Scope:  "global",
				},
			})
		}
	}

	if dc := d.DistributionConfig; dc != nil {
		if dc.Aliases != nil {
			for _, alias := range dc.Aliases.Items {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  alias,
						Scope:  "global",
					},
				})
			}
		}

		if dc.ContinuousDeploymentPolicyId != nil && *dc.ContinuousDeploymentPolicyId != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "cloudfront-continuous-deployment-policy",
					Method: sdp.QueryMethod_GET,
					Query:  *dc.ContinuousDeploymentPolicyId,
					Scope:  scope,
				},
			})
		}

		if dc.CacheBehaviors != nil {
			for _, behavior := range dc.CacheBehaviors.Items {
				if behavior.CachePolicyId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-cache-policy",
							Method: sdp.QueryMethod_GET,
							Query:  *behavior.CachePolicyId,
							Scope:  scope,
						},
					})
				}

				if behavior.FieldLevelEncryptionId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-field-level-encryption",
							Method: sdp.QueryMethod_GET,
							Query:  *behavior.FieldLevelEncryptionId,
							Scope:  scope,
						},
					})
				}

				if behavior.OriginRequestPolicyId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-origin-request-policy",
							Method: sdp.QueryMethod_GET,
							Query:  *behavior.OriginRequestPolicyId,
							Scope:  scope,
						},
					})
				}

				if behavior.RealtimeLogConfigArn != nil {
					if arn, err := ParseARN(*behavior.RealtimeLogConfigArn); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "cloudfront-realtime-log-config",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *behavior.RealtimeLogConfigArn,
								Scope:  FormatScope(arn.AccountID, arn.Region),
							},
						})
					}
				}

				if behavior.ResponseHeadersPolicyId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-response-headers-policy",
							Method: sdp.QueryMethod_GET,
							Query:  *behavior.ResponseHeadersPolicyId,
							Scope:  scope,
						},
					})
				}

				if behavior.TrustedKeyGroups != nil {
					for _, keyGroup := range behavior.TrustedKeyGroups.Items {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "cloudfront-key-group",
								Query:  keyGroup,
								Method: sdp.QueryMethod_GET,
								Scope:  scope,
							},
						})
					}
				}

				if behavior.FunctionAssociations != nil {
					for _, function := range behavior.FunctionAssociations.Items {
						if function.FunctionARN != nil {
							if arn, err := ParseARN(*function.FunctionARN); err == nil {
								item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
									Query: &sdp.Query{
										Type:   "cloudfront-function",
										Method: sdp.QueryMethod_SEARCH,
										Query:  *function.FunctionARN,
										Scope:  FormatScope(arn.AccountID, arn.Region),
									},
								})
							}
						}
					}
				}

				if behavior.LambdaFunctionAssociations != nil {
					for _, function := range behavior.LambdaFunctionAssociations.Items {
						if arn, err := ParseARN(*function.LambdaFunctionARN); err == nil {
							item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   "lambda-function",
									Method: sdp.QueryMethod_SEARCH,
									Query:  *function.LambdaFunctionARN,
									Scope:  FormatScope(arn.AccountID, arn.Region),
								},
							})
						}
					}
				}
			}
		}

		if dc.Origins != nil {
			for _, origin := range dc.Origins.Items {
				if origin.DomainName != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *origin.DomainName,
							Scope:  "global",
						},
					})
				}

				if origin.OriginAccessControlId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-origin-access-control",
							Method: sdp.QueryMethod_GET,
							Query:  *origin.OriginAccessControlId,
							Scope:  scope,
						},
					})
				}

				if origin.S3OriginConfig != nil {
					// If this is set then the origin is an S3 bucket, so we can
					// try to get the bucket name from the domain name
					if origin.DomainName != nil {
						matches := s3DnsRegex.FindStringSubmatch(*origin.DomainName)

						if len(matches) == 3 {
							item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   "s3-bucket",
									Method: sdp.QueryMethod_GET,
									Query:  matches[1],
									Scope:  FormatScope(scope, ""), // S3 buckets are global
								},
							})
						}
					}

					if origin.S3OriginConfig.OriginAccessIdentity != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "cloudfront-cloud-front-origin-access-identity",
								Method: sdp.QueryMethod_GET,
								Query:  *origin.S3OriginConfig.OriginAccessIdentity,
								Scope:  scope,
							},
						})
					}
				}
			}
		}

		if dc.DefaultCacheBehavior != nil {
			if dc.DefaultCacheBehavior.CachePolicyId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-cache-policy",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.DefaultCacheBehavior.CachePolicyId,
						Scope:  scope,
					},
				})
			}

			if dc.DefaultCacheBehavior.FieldLevelEncryptionId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-field-level-encryption",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.DefaultCacheBehavior.FieldLevelEncryptionId,
						Scope:  scope,
					},
				})
			}

			if dc.DefaultCacheBehavior.OriginRequestPolicyId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-origin-request-policy",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.DefaultCacheBehavior.OriginRequestPolicyId,
						Scope:  scope,
					},
				})
			}

			if dc.DefaultCacheBehavior.RealtimeLogConfigArn != nil {
				if arn, err := ParseARN(*dc.DefaultCacheBehavior.RealtimeLogConfigArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-realtime-log-config",
							Method: sdp.QueryMethod_GET,
							Query:  *dc.DefaultCacheBehavior.RealtimeLogConfigArn,
							Scope:  FormatScope(arn.AccountID, arn.Region),
						},
					})
				}
			}

			if dc.DefaultCacheBehavior.ResponseHeadersPolicyId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-response-headers-policy",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.DefaultCacheBehavior.ResponseHeadersPolicyId,
						Scope:  scope,
					},
				})
			}

			if dc.DefaultCacheBehavior.TrustedKeyGroups != nil {
				for _, keyGroup := range dc.DefaultCacheBehavior.TrustedKeyGroups.Items {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "cloudfront-key-group",
							Query:  keyGroup,
							Method: sdp.QueryMethod_GET,
							Scope:  scope,
						},
					})
				}
			}

			if dc.DefaultCacheBehavior.FunctionAssociations != nil {
				for _, function := range dc.DefaultCacheBehavior.FunctionAssociations.Items {
					if function.FunctionARN != nil {
						if arn, err := ParseARN(*function.FunctionARN); err == nil {
							item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   "cloudfront-function",
									Method: sdp.QueryMethod_SEARCH,
									Query:  *function.FunctionARN,
									Scope:  FormatScope(arn.AccountID, arn.Region),
								},
							})
						}
					}
				}
			}

			if dc.DefaultCacheBehavior.LambdaFunctionAssociations != nil {
				for _, function := range dc.DefaultCacheBehavior.LambdaFunctionAssociations.Items {
					if arn, err := ParseARN(*function.LambdaFunctionARN); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "lambda-function",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *function.LambdaFunctionARN,
								Scope:  FormatScope(arn.AccountID, arn.Region),
							},
						})
					}
				}
			}
		}

		if dc.Logging != nil && dc.Logging.Bucket != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *dc.Logging.Bucket,
					Scope:  "global",
				},
			})
		}

		if dc.ViewerCertificate != nil {
			if dc.ViewerCertificate.ACMCertificateArn != nil {
				if arn, err := ParseARN(*dc.ViewerCertificate.ACMCertificateArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "acm-certificate",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *dc.ViewerCertificate.ACMCertificateArn,
							Scope:  FormatScope(arn.AccountID, arn.Region),
						},
					})
				}
			}
			if dc.ViewerCertificate.IAMCertificateId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-server-certificate",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.ViewerCertificate.IAMCertificateId,
						Scope:  scope,
					},
				})
			}
		}

		if dc.WebACLId != nil {
			if arn, err := ParseARN(*dc.WebACLId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "wafv2-web-acl",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *dc.WebACLId,
						Scope:  FormatScope(arn.AccountID, arn.Region),
					},
				})
			} else {
				// Else assume it's a V1 ID
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "waf-web-acl",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.WebACLId,
						Scope:  scope,
					},
				})
			}
		}
	}

	return &item, nil
}

func NewCloudfrontDistributionAdapter(client CloudFrontClient, accountID string, cache sdpcache.Cache) *AlwaysGetAdapter[*cloudfront.ListDistributionsInput, *cloudfront.ListDistributionsOutput, *cloudfront.GetDistributionInput, *cloudfront.GetDistributionOutput, CloudFrontClient, *cloudfront.Options] {
	return &AlwaysGetAdapter[*cloudfront.ListDistributionsInput, *cloudfront.ListDistributionsOutput, *cloudfront.GetDistributionInput, *cloudfront.GetDistributionOutput, CloudFrontClient, *cloudfront.Options]{
		ItemType:        "cloudfront-distribution",
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: distributionAdapterMetadata,
		cache:           cache,
		Region:          "", // Cloudfront resources aren't tied to a region
		ListInput:       &cloudfront.ListDistributionsInput{},
		ListFuncPaginatorBuilder: func(client CloudFrontClient, input *cloudfront.ListDistributionsInput) Paginator[*cloudfront.ListDistributionsOutput, *cloudfront.Options] {
			return cloudfront.NewListDistributionsPaginator(client, input)
		},
		GetInputMapper: func(scope, query string) *cloudfront.GetDistributionInput {
			return &cloudfront.GetDistributionInput{
				Id: &query,
			}
		},
		ListFuncOutputMapper: func(output *cloudfront.ListDistributionsOutput, input *cloudfront.ListDistributionsInput) ([]*cloudfront.GetDistributionInput, error) {
			var inputs []*cloudfront.GetDistributionInput

			for _, distribution := range output.DistributionList.Items {
				inputs = append(inputs, &cloudfront.GetDistributionInput{
					Id: distribution.Id,
				})
			}

			return inputs, nil
		},
		GetFunc: distributionGetFunc,
	}
}

var distributionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-distribution",
	DescriptiveName: "CloudFront Distribution",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Search:            true,
		Get:               true,
		List:              true,
		GetDescription:    "Get a distribution by ID",
		ListDescription:   "List all distributions",
		SearchDescription: "Search distributions by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_cloudfront_distribution.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{
		"cloudfront-key-group",
		"cloudfront-cloud-front-origin-access-identity",
		"cloudfront-continuous-deployment-policy",
		"cloudfront-cache-policy",
		"cloudfront-field-level-encryption",
		"cloudfront-function",
		"cloudfront-origin-request-policy",
		"cloudfront-realtime-log-config",
		"cloudfront-response-headers-policy",
		"dns",
		"lambda-function",
		"s3-bucket",
	},
})
