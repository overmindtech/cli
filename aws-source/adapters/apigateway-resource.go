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

func convertGetResourceOutputToResource(output *apigateway.GetResourceOutput) *types.Resource {
	return &types.Resource{
		Id:              output.Id,
		ParentId:        output.ParentId,
		Path:            output.Path,
		PathPart:        output.PathPart,
		ResourceMethods: output.ResourceMethods,
	}
}

// query: rest-api-id/resource-id for get request
// query: rest-api-id for search request
func resourceOutputMapper(query, scope string, awsItem *types.Resource) (*sdp.Item, error) {
	var restApiID string

	f := strings.Split(query, "/")

	switch len(f) {
	case 1, 2:
		restApiID = f[0]
	default:
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/resource-id or rest-api-id, but found: %s", query),
		}
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	err = attributes.Set("UniqueName", fmt.Sprintf("%s/%s", restApiID, *awsItem.Id))
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-resource",
		UniqueAttribute: "UniqueName",
		Attributes:      attributes,
		Scope:           scope,
	}

	for methodString := range awsItem.ResourceMethods {
		if awsItem.Id != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "apigateway-method",
					Method: sdp.QueryMethod_GET,
					Query:  fmt.Sprintf("%s/%s/%s", restApiID, *awsItem.Id, methodString),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	return &item, nil
}

func NewAPIGatewayResourceAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.Resource, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.Resource, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-resource",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: apiGatewayResourceAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.Resource, error) {
			f := strings.Split(query, "/")
			if len(f) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/resource-id, but found: %s", query),
				}
			}

			out, err := client.GetResource(ctx, &apigateway.GetResourceInput{
				RestApiId:  &f[0], // rest-api-id
				ResourceId: &f[1], // resource-id
			})
			if err != nil {
				return nil, err
			}

			return convertGetResourceOutputToResource(out), nil
		},
		DisableList: true,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.Resource, error) {
			out, err := client.GetResources(ctx, &apigateway.GetResourcesInput{
				RestApiId: &query,
			})
			if err != nil {
				return nil, err
			}

			var resources []*types.Resource
			for _, resource := range out.Items {
				resources = append(resources, &resource)
			}

			return resources, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.Resource) (*sdp.Item, error) {
			return resourceOutputMapper(query, scope, awsItem)
		},
	}
}

var apiGatewayResourceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-resource",
	DescriptiveName: "API Gateway",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Resource by rest-api-id/resource-id",
		SearchDescription: "Search Resources by REST API ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_resource.id"},
	},
	PotentialLinks: []string{
		"apigateway-method",
	},
})
