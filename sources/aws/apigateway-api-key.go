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
	APIGWAPIKey = shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.APIKey)
)

// apiGatewayKeyWrapper is a struct that wraps the AWS API Gateway API Key functionality
type apiGatewayKeyWrapper struct {
	client *apigateway.Client

	*Base
}

// NewApiGatewayAPIKey creates a new apiGatewayKeyWrapper for AWS API Gateway API Key
func NewApiGatewayAPIKey(client *apigateway.Client, accountID, region string) sources.SearchableListableWrapper {
	return &apiGatewayKeyWrapper{
		client: client,
		Base: NewBase(
			accountID,
			region,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			APIGWAPIKey,
		),
	}
}

// TerraformMappings returns the Terraform mappings for the API Key
func (d *apiGatewayKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "aws_api_gateway_api_key.id",
		},
	}
}

// GetLookups returns the ItemTypeLookups for the Get operation
func (d *apiGatewayKeyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		APIGWAPIKeyLookupByID,
	}
}

// Get retrieves an API Key by its ID and converts it to an sdp.Item
func (d *apiGatewayKeyWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	out, err := d.client.GetApiKey(ctx, &apigateway.GetApiKeyInput{
		ApiKey: &queryParts[0],
	})
	if err != nil {
		return nil, queryError(err)
	}

	return d.awsToSdpItem(convertGetApiKeyOutputToApiKey(out))
}

// SearchLookups returns the ItemTypeLookups for the Search operation
func (d *apiGatewayKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			APIGWAPIKeyLookupByName,
		},
	}
}

// Search retrieves API Keys by a search query and converts them to sdp.Items
func (d *apiGatewayKeyWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	out, err := d.client.GetApiKeys(ctx, &apigateway.GetApiKeysInput{
		NameQuery: &queryParts[0],
	})
	if err != nil {
		return nil, queryError(err)
	}

	return d.mapper(out.Items)
}

// List retrieves all API Keys and converts them to sdp.Items
func (d *apiGatewayKeyWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	out, err := d.client.GetApiKeys(ctx, &apigateway.GetApiKeysInput{})
	if err != nil {
		return nil, queryError(err)
	}

	return d.mapper(out.Items)
}

// mapper converts a list of AWS API Keys to a list of sdp.Items
func (d *apiGatewayKeyWrapper) mapper(apiKeys []types.ApiKey) ([]*sdp.Item, *sdp.QueryError) {
	var items []*sdp.Item

	for _, apiKey := range apiKeys {
		sdpItem, err := d.awsToSdpItem(apiKey)
		if err != nil {
			return nil, err
		}

		items = append(items, sdpItem)
	}

	return items, nil
}

func (d *apiGatewayKeyWrapper) awsToSdpItem(apiKey types.ApiKey) (*sdp.Item, *sdp.QueryError) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(apiKey, "tags")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	item := &sdp.Item{
		Type:            d.Type(),
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           d.Scopes()[0],
		Tags:            apiKey.Tags,
	}

	for _, key := range apiKey.StageKeys {
		// {restApiId}/{stage}
		if sections := strings.Split(key, "/"); len(sections) == 2 {
			restAPIID := sections[0]
			if restAPIID != "" {
				linkedItem := shared.NewItemType(awsshared.AWS, awsshared.APIGateway, awsshared.RESTAPI)
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   linkedItem.String(),
						Method: sdp.QueryMethod_GET,
						Query:  restAPIID,
						Scope:  d.Scopes()[0],
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	return item, nil
}

// convertGetApiKeyOutputToApiKey converts a GetApiKeyOutput to an ApiKey
func convertGetApiKeyOutputToApiKey(output *apigateway.GetApiKeyOutput) types.ApiKey {
	return types.ApiKey{
		Id:              output.Id,
		Name:            output.Name,
		Enabled:         output.Enabled,
		CreatedDate:     output.CreatedDate,
		LastUpdatedDate: output.LastUpdatedDate,
		StageKeys:       output.StageKeys,
		Tags:            output.Tags,
	}
}
