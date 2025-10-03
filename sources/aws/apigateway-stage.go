package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	awsshared "github.com/overmindtech/cli/sources/aws/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	APIGWRestAPI    = shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.RESTAPI)
	APIGWStage      = shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.Stage)
	WAFv2WebACL     = shared.NewItemType(awsshared.AWS, awsshared.WAFv2, awsshared.WebACL)
	APIGWDeployment = shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.Deployment)

	APIGWRestAPILookupByID      = shared.NewItemTypeLookup("id", APIGWRestAPI)
	APIGWDeploymentLookupByName = shared.NewItemTypeLookup("name", APIGWDeployment)
	APIGWStageLookupByName      = shared.NewItemTypeLookup("name", APIGWStage)
	APIGWAPIKeyLookupByID       = shared.NewItemTypeLookup("id", APIGWAPIKey)
	APIGWAPIKeyLookupByName     = shared.NewItemTypeLookup("name", APIGWAPIKey)
)

// apiGatewayKeyWrapper is a struct that wraps the AWS API Gateway Stage functionality
type apiGatewayStageWrapper struct {
	client *apigateway.Client

	*Base
}

// NewAPIGatewayStage creates a new apiGatewayKeyWrapper for AWS API Gateway Stage
func NewAPIGatewayStage(client *apigateway.Client, accountID, region string) sources.SearchableWrapper {
	return &apiGatewayStageWrapper{
		client: client,
		Base: NewBase(
			accountID,
			region,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			APIGWStage),
	}
}

func (d *apiGatewayStageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(WAFv2WebACL)
}

// TerraformMappings returns the Terraform mappings for the Stage
func (d *apiGatewayStageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "aws_api_gateway_stage.id",
		},
	}
}

func (d *apiGatewayStageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		APIGWRestAPILookupByID,
		APIGWStageLookupByName,
	}
}

func (d *apiGatewayStageWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	out, err := d.client.GetStage(ctx, &apigateway.GetStageInput{
		RestApiId: &queryParts[0],
		StageName: &queryParts[1],
	})
	if err != nil {
		return nil, queryError(err)
	}

	return d.awsToSdpItem(convertGetStageOutputToStage(out), queryParts[0])
}

// SearchLookups returns the ItemTypeLookups for the Search operation
func (d *apiGatewayStageWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			APIGWRestAPILookupByID,
		},
		{
			APIGWRestAPILookupByID,
			APIGWDeploymentLookupByName,
		},
	}
}

// Search retrieves Stages by a search query and converts them to sdp.Items
func (d *apiGatewayStageWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	var input *apigateway.GetStagesInput

	switch len(queryParts) {
	case 1:
		input = &apigateway.GetStagesInput{
			RestApiId: &queryParts[0],
		}
	case 2:
		input = &apigateway.GetStagesInput{
			RestApiId:    &queryParts[0],
			DeploymentId: &queryParts[1],
		}
	}

	out, err := d.client.GetStages(ctx, input)
	if err != nil {
		return nil, queryError(err)
	}

	return d.mapper(out.Item, queryParts[0])
}

// mapper converts a list of AWS Stages to a list of sdp.Items
func (d *apiGatewayStageWrapper) mapper(stages []types.Stage, query string) ([]*sdp.Item, *sdp.QueryError) {
	var items []*sdp.Item

	for _, stage := range stages {
		sdpItem, err := d.awsToSdpItem(stage, query)
		if err != nil {
			return nil, err
		}

		items = append(items, sdpItem)
	}

	return items, nil
}

func (d *apiGatewayStageWrapper) awsToSdpItem(stage types.Stage, query string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(stage, "tags")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	restAPIID := strings.Split(query, "/")[0]

	err = attributes.Set("UniqueAttribute", query)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	item := &sdp.Item{
		Type:            d.Type(),
		UniqueAttribute: "StageName",
		Attributes:      attributes,
		Scope:           d.Scopes()[0],
		Tags:            stage.Tags,
	}

	if stage.DeploymentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   APIGWDeployment.String(),
				Method: sdp.QueryMethod_GET,
				Query:  restAPIID + "/" + *stage.DeploymentId,
				Scope:  d.Scopes()[0],
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}

	linkedItemRestAPI := shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.RESTAPI)
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   linkedItemRestAPI.String(),
			Method: sdp.QueryMethod_GET,
			Query:  restAPIID,
			Scope:  d.Scopes()[0],
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	return item, nil
}

// convertGetStageOutputToStage converts a GetStageOutput to a Stage
func convertGetStageOutputToStage(output *apigateway.GetStageOutput) types.Stage {
	return types.Stage{
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

func (d *apiGatewayStageWrapper) PredefinedRole() string {
	// TODO: https://linear.app/overmind/issue/ENG-1526/ensure-the-manual-adapter-framework-is-cloud-provider-agnostic
	return ""
}
