package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func apiGatewayMethodResponseGetFunc(ctx context.Context, client apigatewayClient, scope string, input *apigateway.GetMethodResponseInput) (*sdp.Item, error) {
	if input == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "query must be in the format of: the rest-api-id/resource-id/http-method/status-code",
		}
	}

	output, err := client.GetMethodResponse(ctx, input)
	if err != nil {
		return nil, err
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(output, "tags")
	if err != nil {
		return nil, err
	}

	// We create a custom ID of {rest-api-id}/{resource-id}/{http-method}/{status-code} e.g.
	// rest-api-id/resource-id/GET/200
	methodResponseID := fmt.Sprintf(
		"%s/%s/%s/%s",
		*input.RestApiId,
		*input.ResourceId,
		*input.HttpMethod,
		*input.StatusCode,
	)
	err = attributes.Set("MethodResponseID", methodResponseID)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "apigateway-method-response",
		UniqueAttribute: "MethodResponseID",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-method",
			Method: sdp.QueryMethod_GET,
			Query:  fmt.Sprintf("%s/%s/%s", *input.RestApiId, *input.ResourceId, *input.HttpMethod),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly coupled
			In:  true,
			Out: true,
		},
	})

	return item, nil
}

func NewAPIGatewayMethodResponseAdapter(client apigatewayClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*apigateway.GetMethodResponseInput, *apigateway.GetMethodResponseOutput, *apigateway.GetMethodResponseInput, *apigateway.GetMethodResponseOutput, apigatewayClient, *apigateway.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*apigateway.GetMethodResponseInput, *apigateway.GetMethodResponseOutput, *apigateway.GetMethodResponseInput, *apigateway.GetMethodResponseOutput, apigatewayClient, *apigateway.Options]{
		ItemType:        "apigateway-method-response",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiGatewayMethodResponseAdapterMetadata,
		GetFunc:         apiGatewayMethodResponseGetFunc,
		GetInputMapper: func(scope, query string) *apigateway.GetMethodResponseInput {
			// We are using a custom id of {rest-api-id}/{resource-id}/{http-method}/{status-code} e.g.
			// rest-api-id/resource-id/GET/200
			f := strings.Split(query, "/")
			if len(f) != 4 {
				return nil
			}

			return &apigateway.GetMethodResponseInput{
				RestApiId:  &f[0],
				ResourceId: &f[1],
				HttpMethod: &f[2],
				StatusCode: &f[3],
			}
		},
		DisableList: true,
	}
}

var apiGatewayMethodResponseAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-method-response",
	DescriptiveName: "API Gateway Method Response",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get a Method Response by it's ID: {rest-api-id}/{resource-id}/{http-method}/{status-code}",
		Search:            true,
		SearchDescription: "Search Method Responses by ARN",
	},
})
