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

func convertGetModelOutputToModel(output *apigateway.GetModelOutput) *types.Model {
	return &types.Model{
		Id:          output.Id,
		Name:        output.Name,
		Description: output.Description,
		Schema:      output.Schema,
		ContentType: output.ContentType,
	}
}

func modelOutputMapper(query, scope string, awsItem *types.Model) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	restAPIID := strings.Split(query, "/")[0]

	err = attributes.Set("UniqueAttribute", fmt.Sprintf("%s/%s", restAPIID, *awsItem.Name))
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-model",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-rest-api",
			Method: sdp.QueryMethod_GET,
			Query:  restAPIID,
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

func NewAPIGatewayModelAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.Model, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.Model, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-model",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: modelAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.Model, error) {
			f := strings.Split(query, "/")
			if len(f) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/model-name, but found: %s", query),
				}
			}
			out, err := client.GetModel(ctx, &apigateway.GetModelInput{
				RestApiId: &f[0],
				ModelName: &f[1],
			})
			if err != nil {
				return nil, err
			}
			return convertGetModelOutputToModel(out), nil
		},
		DisableList: true,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.Model, error) {
			out, err := client.GetModels(ctx, &apigateway.GetModelsInput{
				RestApiId: &query,
			})
			if err != nil {
				return nil, err
			}

			var items []*types.Model
			for _, model := range out.Items {
				items = append(items, &model)
			}

			return items, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.Model) (*sdp.Item, error) {
			return modelOutputMapper(query, scope, awsItem)
		},
	}
}

var modelAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-model",
	DescriptiveName: "API Gateway Model",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an API Gateway Model by its rest API ID and model name: rest-api-id/model-name",
		SearchDescription: "Search for API Gateway Models by their rest API ID: rest-api-id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_model.id"},
	},
})
