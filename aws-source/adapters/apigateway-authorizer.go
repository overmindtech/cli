package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// convertGetAuthorizerOutputToAuthorizer converts a GetAuthorizerOutput to an Authorizer
func convertGetAuthorizerOutputToAuthorizer(output *apigateway.GetAuthorizerOutput) *types.Authorizer {
	return &types.Authorizer{
		Id:                           output.Id,
		Name:                         output.Name,
		Type:                         output.Type,
		ProviderARNs:                 output.ProviderARNs,
		AuthType:                     output.AuthType,
		AuthorizerUri:                output.AuthorizerUri,
		AuthorizerCredentials:        output.AuthorizerCredentials,
		IdentitySource:               output.IdentitySource,
		IdentityValidationExpression: output.IdentityValidationExpression,
		AuthorizerResultTtlInSeconds: output.AuthorizerResultTtlInSeconds,
	}
}

func authorizerOutputMapper(query, scope string, awsItem *types.Authorizer) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-authorizer",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-rest-api",
			Method: sdp.QueryMethod_GET,
			Query:  strings.Split(query, "/")[0],
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// They are tightly coupled, so we need to propagate the blast to the linked item
			In:  true,
			Out: true,
		},
	})

	return &item, nil
}

func NewAPIGatewayAuthorizerAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.Authorizer, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.Authorizer, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-authorizer",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: authorizerAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.Authorizer, error) {
			f := strings.Split(query, "/")
			if len(f) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/authorizer-id, but found: %s", query),
				}
			}
			out, err := client.GetAuthorizer(ctx, &apigateway.GetAuthorizerInput{
				RestApiId:    &f[0],
				AuthorizerId: &f[1],
			})
			if err != nil {
				return nil, err
			}
			return convertGetAuthorizerOutputToAuthorizer(out), nil
		},
		DisableList: true,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.Authorizer, error) {
			out, err := client.GetAuthorizers(ctx, &apigateway.GetAuthorizersInput{
				RestApiId: &query,
			})
			if err != nil {
				return nil, err
			}

			authorizers := make([]*types.Authorizer, 0, len(out.Items))
			for _, authorizer := range out.Items {
				authorizers = append(authorizers, &authorizer)
			}

			return authorizers, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.Authorizer) (*sdp.Item, error) {
			return authorizerOutputMapper(query, scope, awsItem)
		},
	}
}

var authorizerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-authorizer",
	DescriptiveName: "API Gateway Authorizer",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an API Gateway Authorizer by its rest API ID and ID: rest-api-id/authorizer-id",
		SearchDescription: "Search for API Gateway Authorizers by their rest API ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_authorizer.id"},
	},
})
