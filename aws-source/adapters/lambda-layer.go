package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func layerListFunc(ctx context.Context, client *lambda.Client, scope string) ([]*types.LayersListItem, error) {
	paginator := lambda.NewListLayersPaginator(client, &lambda.ListLayersInput{})
	layers := make([]*types.LayersListItem, 0)

	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)

		if err != nil {
			return nil, err
		}

		for _, layer := range out.Layers {
			layers = append(layers, &layer)
		}
	}

	return layers, nil
}

func layerItemMapper(_, scope string, awsItem *types.LayersListItem) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "lambda-layer",
		UniqueAttribute: "LayerName",
		Attributes:      attributes,
		Scope:           scope,
	}

	if awsItem.LatestMatchingVersion != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "lambda-layer-version",
				Method: sdp.QueryMethod_GET,
				Query:  fmt.Sprintf("%v:%v", *awsItem.LayerName, awsItem.LatestMatchingVersion.Version),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Tightly coupled
				In:  true,
				Out: true,
			},
		})
	}

	return &item, nil
}

func NewLambdaLayerAdapter(client *lambda.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.LayersListItem, *lambda.Client, *lambda.Options] {
	return &adapterhelpers.GetListAdapter[*types.LayersListItem, *lambda.Client, *lambda.Options]{
		ItemType:        "lambda-layer",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: lambdaLayerAdapterMetadata,
		GetFunc: func(_ context.Context, _ *lambda.Client, _, _ string) (*types.LayersListItem, error) {
			// Layers can only be listed
			return nil, errors.New("get is not supported for lambda-layers")
		},
		ListFunc:   layerListFunc,
		ItemMapper: layerItemMapper,
	}
}

var lambdaLayerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "lambda-layer",
	DescriptiveName: "Lambda Layer",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		List:            true,
		ListDescription: "List all lambda layers",
	},
	PotentialLinks: []string{"lambda-layer-version"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
