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

func convertGetStageOutputToStage(output *apigateway.GetStageOutput) *types.Stage {
	return &types.Stage{
		DeploymentId:         output.DeploymentId,
		StageName:            output.StageName,
		Description:          output.Description,
		CreatedDate:          output.CreatedDate,
		LastUpdatedDate:      output.LastUpdatedDate,
		Variables:            output.Variables,
		AccessLogSettings:    output.AccessLogSettings,
		CacheClusterEnabled:  output.CacheClusterEnabled,
		CacheClusterSize:     output.CacheClusterSize,
		CacheClusterStatus:   output.CacheClusterStatus,
		CanarySettings:       output.CanarySettings,
		ClientCertificateId:  output.ClientCertificateId,
		DocumentationVersion: output.DocumentationVersion,
		MethodSettings:       output.MethodSettings,
		TracingEnabled:       output.TracingEnabled,
		WebAclArn:            output.WebAclArn,
		Tags:                 output.Tags,
	}
}

func stageOutputMapper(query, scope string, awsItem *types.Stage) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem, "tags")
	if err != nil {
		return nil, err
	}

	// if it is `GET`, the query will be: rest-api-id/stage-name
	// if it is `SEARCH`, the query will be: rest-api-id/deployment-id or rest-api-id
	restAPIID := strings.Split(query, "/")[0]

	err = attributes.Set("UniqueAttribute", fmt.Sprintf("%s/%s", restAPIID, *awsItem.StageName))
	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "apigateway-stage",
		UniqueAttribute: "StageName",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            awsItem.Tags,
	}

	if awsItem.DeploymentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "apigateway-deployment",
				Method: sdp.QueryMethod_GET,
				Query:  fmt.Sprintf("%s/%s", restAPIID, *awsItem.DeploymentId),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Deleting a deployment will impact the stage
				In: true,
				// Deleting a stage won't impact the deployment
				Out: false,
			},
		})
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

func NewAPIGatewayStageAdapter(client *apigateway.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.Stage, *apigateway.Client, *apigateway.Options] {
	return &adapterhelpers.GetListAdapter[*types.Stage, *apigateway.Client, *apigateway.Options]{
		ItemType:        "apigateway-stage",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: stageAdapterMetadata,
		GetFunc: func(ctx context.Context, client *apigateway.Client, scope, query string) (*types.Stage, error) {
			f := strings.Split(query, "/")
			if len(f) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the rest-api-id/stage-name, but found: %s", query),
				}
			}
			out, err := client.GetStage(ctx, &apigateway.GetStageInput{
				RestApiId: &f[0],
				StageName: &f[1],
			})
			if err != nil {
				return nil, err
			}
			return convertGetStageOutputToStage(out), nil
		},
		DisableList: true,
		SearchFunc: func(ctx context.Context, client *apigateway.Client, scope string, query string) ([]*types.Stage, error) {
			f := strings.Split(query, "/")
			var input *apigateway.GetStagesInput

			switch len(f) {
			case 1:
				input = &apigateway.GetStagesInput{
					RestApiId: &f[0],
				}
			case 2:
				input = &apigateway.GetStagesInput{
					RestApiId:    &f[0],
					DeploymentId: &f[1],
				}
			default:
				return nil, &sdp.QueryError{
					ErrorType: sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf(
						"query must be in the format of: the rest-api-id/deployment-id or rest-api-id, but found: %s",
						query,
					),
				}
			}

			out, err := client.GetStages(ctx, input)
			if err != nil {
				return nil, err
			}

			var items []*types.Stage
			for _, stage := range out.Item {
				items = append(items, &stage)
			}

			return items, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.Stage) (*sdp.Item, error) {
			return stageOutputMapper(query, scope, awsItem)
		},
	}
}

var stageAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "apigateway-stage",
	DescriptiveName: "API Gateway Stage",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an API Gateway Stage by its rest API ID and stage name: rest-api-id/stage-name",
		SearchDescription: "Search for API Gateway Stages by their rest API ID or with rest API ID and deployment-id: rest-api-id/deployment-id",
	},
	PotentialLinks: []string{"wafv2-web-acl"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_api_gateway_stage.id"},
	},
})
