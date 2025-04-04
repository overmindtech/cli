package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func directconnectConnectionOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeConnectionsInput, output *directconnect.DescribeConnectionsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, connection := range output.Connections {
		attributes, err := adapterhelpers.ToAttributesWithExclude(connection, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-connection",
			UniqueAttribute: "ConnectionId",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            directconnectTagsToMap(connection.Tags),
		}

		if connection.LagId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-lag",
					Method: sdp.QueryMethod_GET,
					Query:  *connection.LagId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Connection and LAG are tightly coupled
					// Changing one will affect the other
					In:  true,
					Out: true,
				},
			})
		}

		if connection.Location != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-location",
					Method: sdp.QueryMethod_GET,
					Query:  *connection.Location,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the location will affect this, i.e., its speed, provider, etc.
					In: true,
					// We can't affect the location
					Out: false,
				},
			})
		}

		if connection.LoaIssueTime != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-loa",
					Method: sdp.QueryMethod_GET,
					Query:  *connection.ConnectionId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the loa will affect this
					In: true,
					// We can't affect the loa
					Out: false,
				},
			})
		}

		// Virtual Interfaces
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "directconnect-virtual-interface",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *connection.ConnectionId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the virtual interface won't affect this
				In: false,
				// We cannot delete a connection if it has virtual interfaces
				Out: true,
			},
		})

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectConnectionAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeConnectionsInput, *directconnect.DescribeConnectionsOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeConnectionsInput, *directconnect.DescribeConnectionsOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-connection",
		AdapterMetadata: directconnectConnectionAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeConnectionsInput) (*directconnect.DescribeConnectionsOutput, error) {
			return client.DescribeConnections(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*directconnect.DescribeConnectionsInput, error) {
			return &directconnect.DescribeConnectionsInput{
				ConnectionId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeConnectionsInput, error) {
			return &directconnect.DescribeConnectionsInput{}, nil
		},
		OutputMapper: directconnectConnectionOutputMapper,
	}
}

var directconnectConnectionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-connection",
	DescriptiveName: "Connection",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a connection by ID",
		ListDescription:   "List all connections",
		SearchDescription: "Search connection by ARN",
	},
	PotentialLinks: []string{"directconnect-lag", "directconnect-location", "directconnect-loa", "directconnect-virtual-interface"},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_dx_connection.id",
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
