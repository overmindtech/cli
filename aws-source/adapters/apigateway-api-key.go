package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// convertGetApiKeyOutputToApiKey converts a GetApiKeyOutput to an ApiKey
func convertGetApiKeyOutputToApiKey(output *apigateway.GetApiKeyOutput) *types.ApiKey {
	return &types.ApiKey{
		Id:              output.Id,
		Name:            output.Name,
		Enabled:         output.Enabled,
		CreatedDate:     output.CreatedDate,
		LastUpdatedDate: output.LastUpdatedDate,
		StageKeys:       output.StageKeys,
		Tags:            output.Tags,
	}
}

func apiKeyListFunc(ctx context.Context, client *apigateway.Client, _ string) ([]*types.ApiKey, error) {
	out, err := client.GetApiKeys(ctx, &apigateway.GetApiKeysInput{})
	if err != nil {
		return nil, err
	}

	var items []*types.ApiKey
	for _, apiKey := range out.Items {
		items = append(items, &apiKey)
	}

	return items, nil
}

func apiKeyOutputMapper(scope string, awsItem *types.ApiKey) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-api-key",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            awsItem.Tags,
	}

	for _, key := range awsItem.StageKeys {
		// {restApiId}/{stage}
		if sections := strings.Split(key, "/"); len(sections) == 2 {
			restAPIID := sections[0]
			if restAPIID != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "apigateway-rest-api",
						Method: sdp.QueryMethod_GET,
						Query:  restAPIID,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// They are tightly coupled, so we need to propagate both ways
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	return &item, nil
}

func NewAPIGatewayApiKeyAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.ApiKey, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.ApiKey, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-api-key",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiKeyAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.ApiKey, error) {
			out, err := client.GetApiKey(ctx, &apigateway.GetApiKeyInput{
				ApiKey: &query,
			})
			if err != nil {
				return nil, err
			}
			return convertGetApiKeyOutputToApiKey(out), nil
		},
		ListFunc: apiKeyListFunc,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.ApiKey, error) {
			out, err := client.GetApiKeys(ctx, &apigateway.GetApiKeysInput{
				NameQuery: &query,
			})
			if err != nil {
				return nil, err
			}

			var items []*types.ApiKey
			for _, apiKey := range out.Items {
				items = append(items, &apiKey)
			}

			return items, nil
		},
		ItemMapper: func(_, scope string, awsItem *types.ApiKey) (*sdp.Item, error) {
			return apiKeyOutputMapper(scope, awsItem)
		},
	}
}

var apiKeyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-api-key",
	DescriptiveName: "API Key",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an API Key by ID",
		ListDescription:   "List all API Keys",
		SearchDescription: "Search for API Keys by their name",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_api_key.id"},
	},
})
