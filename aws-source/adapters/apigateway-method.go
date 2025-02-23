package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type apigatewayClient interface {
	GetMethod(ctx context.Context, params *apigateway.GetMethodInput, optFns ...func(*apigateway.Options)) (*apigateway.GetMethodOutput, error)
	GetMethodResponse(ctx context.Context, params *apigateway.GetMethodResponseInput, optFns ...func(*apigateway.Options)) (*apigateway.GetMethodResponseOutput, error)
}

func apiGatewayMethodGetFunc(ctx context.Context, client apigatewayClient, scope string, input *apigateway.GetMethodInput) (*sdp.Item, error) {
	if input == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "query must be in the format of: the rest-api-id/resource-id/http-method",
		}
	}

	output, err := client.GetMethod(ctx, input)
	if err != nil {
		return nil, err
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(output, "tags")
	if err != nil {
		return nil, err
	}

	// We create a custom ID of {rest-api-id}/{resource-id}/{http-method} e.g.
	// rest-api-id/resource-id/GET
	methodID := fmt.Sprintf(
		"%s/%s/%s",
		*input.RestApiId,
		*input.ResourceId,
		*input.HttpMethod,
	)
	err = attributes.Set("MethodID", methodID)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "apigateway-method",
		UniqueAttribute: "MethodID",
		Attributes:      attributes,
		Scope:           scope,
	}

	if output.MethodIntegration != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-integration",
				Method: sdp.QueryMethod_GET,
				Query:  methodID,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// They are tightly coupled
				In:  true,
				Out: true,
			},
		})
	}

	if output.AuthorizerId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-authorizer",
				Method: sdp.QueryMethod_GET,
				Query:  fmt.Sprintf("%s/%s", *input.RestApiId, *output.AuthorizerId),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Deleting authorizer will affect the method
				In: true,
				// Deleting method won't affect the authorizer
				Out: false,
			},
		})
	}

	if output.RequestValidatorId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-request-validator",
				Method: sdp.QueryMethod_GET,
				Query:  fmt.Sprintf("%s/%s", *input.RestApiId, *output.RequestValidatorId),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Deleting request validator will affect the method
				In: true,
				// Deleting method won't affect the request validator
				Out: false,
			},
		})
	}

	for statusCode := range output.MethodResponses {
		if input.RestApiId != nil && input.ResourceId != nil && input.HttpMethod != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "apigateway-method-response",
					Method: sdp.QueryMethod_GET,
					Query:  fmt.Sprintf("%s/%s/%s/%s", *input.RestApiId, *input.ResourceId, *input.HttpMethod, statusCode),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// They are tightly coupled
					In:  true,
					Out: true,
				},
			})
		}
	}

	return item, nil
}

func NewAPIGatewayMethodAdapter(client apigatewayClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*apigateway.GetMethodInput, *apigateway.GetMethodOutput, *apigateway.GetMethodInput, *apigateway.GetMethodOutput, apigatewayClient, *apigateway.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*apigateway.GetMethodInput, *apigateway.GetMethodOutput, *apigateway.GetMethodInput, *apigateway.GetMethodOutput, apigatewayClient, *apigateway.Options]{
		ItemType:        "apigateway-method",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiGatewayMethodAdapterMetadata,
		GetFunc:         apiGatewayMethodGetFunc,
		GetInputMapper: func(scope, query string) *apigateway.GetMethodInput {
			// We are using a custom id of {rest-api-id}/{resource-id}/{http-method} e.g.
			// rest-api-id/resource-id/GET
			f := strings.Split(query, "/")
			if len(f) != 3 {
				return nil
			}

			return &apigateway.GetMethodInput{
				RestApiId:  &f[0],
				ResourceId: &f[1],
				HttpMethod: &f[2],
			}
		},
		DisableList: true,
	}
}

var apiGatewayMethodAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-method",
	DescriptiveName: "API Gateway Method",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get a Method by it's ID: {rest-api-id}/{resource-id}/{http-method}",
		Search:            true,
		SearchDescription: "Search Methods by ARN",
	},
	PotentialLinks: []string{
		"apigateway-integration",
		"apigateway-authorizer",
		"apigateway-request-validator",
		"apigateway-method-response",
	},
})
