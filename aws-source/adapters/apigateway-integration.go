package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type apiGatewayIntegrationGetter interface {
	GetIntegration(ctx context.Context, params *apigateway.GetIntegrationInput, optFns ...func(*apigateway.Options)) (*apigateway.GetIntegrationOutput, error)
}

func apiGatewayIntegrationGetFunc(ctx context.Context, client apiGatewayIntegrationGetter, scope string, input *apigateway.GetIntegrationInput) (*sdp.Item, error) {
	if input == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "query must be in the format of: rest-api-id/resource-id/http-method",
		}
	}

	output, err := client.GetIntegration(ctx, input)
	if err != nil {
		return nil, err
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(output, "tags")
	if err != nil {
		return nil, err
	}

	// We create a custom ID of {rest-api-id}/{resource-id}/{http-method} e.g.
	// rest-api-id/resource-id/GET
	integrationID := fmt.Sprintf(
		"%s/%s/%s",
		*input.RestApiId,
		*input.ResourceId,
		*input.HttpMethod,
	)
	err = attributes.Set("IntegrationID", integrationID)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "apigateway-integration",
		UniqueAttribute: "IntegrationID",
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

	if output.ConnectionId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-vpc-link",
				Method: sdp.QueryMethod_GET,
				Query:  *output.ConnectionId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// If VPC link goes away, so does the integration
				In: true,
				// If integration goes away, VPC link is still there
				Out: false,
			},
		})
	}

	return item, nil
}

func NewAPIGatewayIntegrationAdapter(client apiGatewayIntegrationGetter, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*apigateway.GetIntegrationInput, *apigateway.GetIntegrationOutput, *apigateway.GetIntegrationInput, *apigateway.GetIntegrationOutput, apiGatewayIntegrationGetter, *apigateway.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*apigateway.GetIntegrationInput, *apigateway.GetIntegrationOutput, *apigateway.GetIntegrationInput, *apigateway.GetIntegrationOutput, apiGatewayIntegrationGetter, *apigateway.Options]{
		ItemType:        "apigateway-integration",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiGatewayIntegrationAdapterMetadata,
		GetFunc:         apiGatewayIntegrationGetFunc,
		GetInputMapper: func(scope, query string) *apigateway.GetIntegrationInput {
			// We are using a custom id of {rest-api-id}/{resource-id}/{http-method} e.g.
			// rest-api-id/resource-id/GET
			f := strings.Split(query, "/")
			if len(f) != 3 {
				slog.Error(
					"query must be in the format of: rest-api-id/resource-id/http-method",
					"found",
					query,
				)

				return nil
			}

			return &apigateway.GetIntegrationInput{
				RestApiId:  &f[0],
				ResourceId: &f[1],
				HttpMethod: &f[2],
			}
		},
		DisableList: true,
	}
}

var apiGatewayIntegrationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-integration",
	DescriptiveName: "API Gateway Integration",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get an Integration by rest-api id, resource id, and http-method",
		Search:            true,
		SearchDescription: "Search Integrations by ARN",
	},
})
