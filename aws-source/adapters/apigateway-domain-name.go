package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func convertGetDomainNameOutputToDomainName(output *apigateway.GetDomainNameOutput) *types.DomainName {
	return &types.DomainName{
		DomainName:                          output.DomainName,
		CertificateArn:                      output.CertificateArn,
		CertificateName:                     output.CertificateName,
		CertificateUploadDate:               output.CertificateUploadDate,
		DistributionDomainName:              output.DistributionDomainName,
		DistributionHostedZoneId:            output.DistributionHostedZoneId,
		RegionalDomainName:                  output.RegionalDomainName,
		RegionalHostedZoneId:                output.RegionalHostedZoneId,
		EndpointConfiguration:               output.EndpointConfiguration,
		DomainNameStatus:                    output.DomainNameStatus,
		DomainNameStatusMessage:             output.DomainNameStatusMessage,
		SecurityPolicy:                      output.SecurityPolicy,
		MutualTlsAuthentication:             output.MutualTlsAuthentication,
		Tags:                                output.Tags,
		OwnershipVerificationCertificateArn: output.OwnershipVerificationCertificateArn,
		RegionalCertificateName:             output.RegionalCertificateName,
		RegionalCertificateArn:              output.RegionalCertificateArn,
	}
}

func domainNameOutputMapper(_, scope string, awsItem *types.DomainName) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-domain-name",
		UniqueAttribute: "DomainName",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            awsItem.Tags,
	}

	// Health based on the DomainNameStatus
	switch awsItem.DomainNameStatus {
	case types.DomainNameStatusAvailable:
		item.Health = sdp.Health_HEALTH_OK.Enum()
	case types.DomainNameStatusUpdating,
		types.DomainNameStatusPending,
		types.DomainNameStatusPendingCertificateReimport,
		types.DomainNameStatusPendingOwnershipVerification:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	default:
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("unknown Domain Name State: %s", awsItem.DomainNameStatus),
		}
	}

	if awsItem.RegionalHostedZoneId != nil {
		//+overmind:link route53-hosted-zone
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "route53-hosted-zone",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.RegionalHostedZoneId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the hosted zone can affect the domain name
				In: true,
				// The domain name won't affect the hosted zone
				Out: false,
			},
		})
	}

	if awsItem.DistributionHostedZoneId != nil {
		//+overmind:link route53-hosted-zone
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "route53-hosted-zone",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.DistributionHostedZoneId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the hosted zone can affect the domain name
				In: true,
				// The domain name won't affect the hosted zone
				Out: false,
			},
		})
	}

	if awsItem.CertificateArn != nil {
		if a, err := adapterhelpers.ParseARN(*awsItem.CertificateArn); err == nil {
			//+overmind:link acm-certificate
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "acm-certificate",
					Method: sdp.QueryMethod_GET,
					Query:  *awsItem.CertificateArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// They are tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	if awsItem.RegionalCertificateArn != nil {
		if a, err := adapterhelpers.ParseARN(*awsItem.RegionalCertificateArn); err == nil {
			//+overmind:link acm-certificate
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "acm-certificate",
					Method: sdp.QueryMethod_GET,
					Query:  *awsItem.RegionalCertificateArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// They are tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	if awsItem.RegionalDomainName != nil {
		//+overmind:link apigateway-domain-name
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-domain-name",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.RegionalDomainName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// They are tightly linked
				In:  true,
				Out: true,
			},
		})
	}

	if awsItem.OwnershipVerificationCertificateArn != nil {
		if a, err := adapterhelpers.ParseARN(*awsItem.OwnershipVerificationCertificateArn); err == nil {
			//+overmind:link acm-certificate
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "acm-certificate",
					Method: sdp.QueryMethod_GET,
					Query:  *awsItem.OwnershipVerificationCertificateArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// They are tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	// TODO: if cloudfront distribution supports searching by name, link it here via awsItem.DistributionDomainName

	return &item, nil
}

func NewAPIGatewayDomainNameAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.DomainName, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.DomainName, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-domain-name",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiGatewayDomainNameAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.DomainName, error) {
			if query == "" {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "query must be the domain-name, but found empty query",
				}
			}

			out, err := client.GetDomainName(ctx, &apigateway.GetDomainNameInput{
				DomainName: &query,
			})
			if err != nil {
				return nil, err
			}

			return convertGetDomainNameOutputToDomainName(out), nil
		},
		ListFunc: func(ctx context.Context, client *apigateway.Client, scope string) ([]*types.DomainName, error) {
			out, err := client.GetDomainNames(ctx, &apigateway.GetDomainNamesInput{})
			if err != nil {
				return nil, err
			}

			var domainNames []*types.DomainName
			for _, domainName := range out.Items {
				domainNames = append(domainNames, &domainName)
			}

			return domainNames, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.DomainName) (*sdp.Item, error) {
			return domainNameOutputMapper(query, scope, awsItem)
		},
	}
}

var apiGatewayDomainNameAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-domain-name",
	DescriptiveName: "API Gateway Domain Name",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get a Domain Name by domain-name",
		Search:            true,
		SearchDescription: "Search Domain Names by ARN",
		List:              true,
		ListDescription:   "List Domain Names",
	},
	PotentialLinks: []string{"acm-certificate"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_domain_name.domain_name"},
	},
})
