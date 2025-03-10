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

// convertGetDeploymentOutputToDeployment converts a GetDeploymentOutput to a Deployment
func convertGetDeploymentOutputToDeployment(output *apigateway.GetDeploymentOutput) *types.Deployment {
	return &types.Deployment{
		Id:          output.Id,
		CreatedDate: output.CreatedDate,
		Description: output.Description,
		ApiSummary:  output.ApiSummary,
	}
}

func deploymentOutputMapper(query, scope string, awsItem *types.Deployment) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-deployment",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
	}

	restAPIID := strings.Split(query, "/")[0]

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

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "apigateway-stage",
			Method: sdp.QueryMethod_SEARCH,
			Query:  fmt.Sprintf("%s/%s", restAPIID, *awsItem.Id),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			/*
				If an aws_api_gateway_deployment is deleted,
				any stage that references this deployment will be affected
				because the stage will no longer have a valid deployment to point to.
				However, if an aws_api_gateway_stage is deleted,
				it does not affect the aws_api_gateway_deployment itself,
				but it will remove the specific environment where the deployment was available.
			*/
			In:  true,
			Out: true,
		},
	})

	return &item, nil
}

func NewAPIGatewayDeploymentAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.Deployment, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.Deployment, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-deployment",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: deploymentAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.Deployment, error) {
			f := strings.Split(query, "/")
			if len(f) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/deployment-id, but found: %s", query),
				}
			}
			out, err := client.GetDeployment(ctx, &apigateway.GetDeploymentInput{
				RestApiId:    &f[0],
				DeploymentId: &f[1],
			})
			if err != nil {
				return nil, err
			}
			return convertGetDeploymentOutputToDeployment(out), nil
		},
		DisableList: true,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.Deployment, error) {
			out, err := client.GetDeployments(ctx, &apigateway.GetDeploymentsInput{
				RestApiId: &query,
			})
			if err != nil {
				return nil, err
			}

			response := make([]*types.Deployment, 0, len(out.Items))
			for _, item := range out.Items {
				response = append(response, &item)
			}

			return response, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.Deployment) (*sdp.Item, error) {
			return deploymentOutputMapper(query, scope, awsItem)
		},
	}
}

var deploymentAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-deployment",
	DescriptiveName: "API Gateway Deployment",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an API Gateway Deployment by its rest API ID and ID: rest-api-id/deployment-id",
		SearchDescription: "Search for API Gateway Deployments by their rest API ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_deployment.id"},
	},
})
