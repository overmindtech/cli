package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"

	log "github.com/sirupsen/logrus"
)

// convertGetRestApiOutputToRestApi converts a GetRestApiOutput to a RestApi
func convertGetRestApiOutputToRestApi(output *apigateway.GetRestApiOutput) *types.RestApi {
	return &types.RestApi{
		CreatedDate:               output.CreatedDate,
		Description:               output.Description,
		Id:                        output.Id,
		Name:                      output.Name,
		Tags:                      output.Tags,
		ApiKeySource:              output.ApiKeySource,
		BinaryMediaTypes:          output.BinaryMediaTypes,
		DisableExecuteApiEndpoint: output.DisableExecuteApiEndpoint,
		EndpointConfiguration:     output.EndpointConfiguration,
		MinimumCompressionSize:    output.MinimumCompressionSize,
		Policy:                    output.Policy,
		RootResourceId:            output.RootResourceId,
		Version:                   output.Version,
		Warnings:                  output.Warnings,
	}
}

func restApiListFunc(ctx context.Context, client *apigateway.Client, _ string) ([]*types.RestApi, error) {
	out, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return nil, err
	}

	var items []*types.RestApi
	for _, restAPI := range out.Items {
		items = append(items, &restAPI)
	}

	return items, nil
}

func restApiOutputMapper(scope string, awsItem *types.RestApi) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	if awsItem.Policy != nil {
		type restAPIWithParsedPolicy struct {
			*types.RestApi
			PolicyDocument *policy.Policy
		}

		restApi := restAPIWithParsedPolicy{
			RestApi: awsItem,
		}

		restApi.PolicyDocument, err = ParsePolicyDocument(*awsItem.Policy)
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"scope":          scope,
				"policyDocument": *awsItem.Policy,
			}).Error("Error parsing policy document")

			return nil, nil //nolint:nilerr
		}

		attributes, err = adapterhelpers.ToAttributesWithExclude(restApi, "tags")
		if err != nil {
			return nil, err
		}
	}

	item := sdp.Item{
		Type:            "apigateway-rest-api",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            awsItem.Tags,
	}

	if awsItem.EndpointConfiguration != nil && awsItem.EndpointConfiguration.VpcEndpointIds != nil {
		for _, vpcEndpointID := range awsItem.EndpointConfiguration.VpcEndpointIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc-endpoint",
					Method: sdp.QueryMethod_GET,
					Query:  vpcEndpointID,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Any change on the VPC endpoint should affect the REST API
					In: true,
					// We can't affect the VPC endpoint
					Out: false,
				},
			})
		}
	}

	if awsItem.RootResourceId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-resource",
				Method: sdp.QueryMethod_GET,
				Query:  fmt.Sprintf("%s/%s", *awsItem.Id, *awsItem.RootResourceId),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// They are tightly linked
				In:  true,
				Out: true,
			},
		})
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-resource",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *awsItem.Id,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Updating a resource won't affect the REST API
			In: false,
			// Updating the REST API will affect the resources
			Out: true,
		},
	})

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-model",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *awsItem.Id,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly linked
			In:  true,
			Out: true,
		},
	})

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-deployment",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *awsItem.Id,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly linked
			In:  false,
			Out: true,
		},
	})

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-authorizer",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *awsItem.Id,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly linked
			In:  true,
			Out: true,
		},
	})

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-stage",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *awsItem.Id,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly linked
			In:  true,
			Out: true,
		},
	})

	return &item, nil
}

func NewAPIGatewayRestApiAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.RestApi, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.RestApi, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-rest-api",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: restApiAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.RestApi, error) {
			out, err := client.GetRestApi(ctx, &apigateway.GetRestApiInput{
				RestApiId: &query,
			})
			if err != nil {
				return nil, err
			}
			return convertGetRestApiOutputToRestApi(out), nil
		},
		ListFunc: restApiListFunc,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.RestApi, error) {
			out, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
			if err != nil {
				return nil, err
			}

			var items []*types.RestApi
			for _, restAPI := range out.Items {
				if *restAPI.Name == query {
					items = append(items, &restAPI)
				}
			}

			return items, nil
		},
		ItemMapper: func(_, scope string, awsItem *types.RestApi) (*sdp.Item, error) {
			return restApiOutputMapper(scope, awsItem)
		},
	}
}

var restApiAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-rest-api",
	DescriptiveName: "REST API",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a REST API by ID",
		ListDescription:   "List all REST APIs",
		SearchDescription: "Search for REST APIs by their name",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_rest_api.id"},
	},
	PotentialLinks: []string{"ec2-vpc-endpoint", "apigateway-resource"},
})
