package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

const (
	directConnectGatewayIDVirtualGatewayIDFormat = "direct_connect_gateway_id/virtual_gateway_id"
	virtualGatewayIDFormat                       = "virtual_gateway_id"
)

func directConnectGatewayAssociationOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeDirectConnectGatewayAssociationsInput, output *directconnect.DescribeDirectConnectGatewayAssociationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, association := range output.DirectConnectGatewayAssociations {
		attributes, err := adapterhelpers.ToAttributesWithExclude(association, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-direct-connect-gateway-association",
			UniqueAttribute: "AssociationId",
			Attributes:      attributes,
			Scope:           scope,
		}

		// stateChangeError =>The error message if the state of an object failed to advance.
		if association.StateChangeError != nil {
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		} else {
			item.Health = sdp.Health_HEALTH_OK.Enum()
		}

		if association.DirectConnectGatewayId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *association.DirectConnectGatewayId,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Deleting a direct connect gateway will change the state of the association
					In: true,
					// We can't affect the direct connect gateway
					Out: false,
				},
			})
		}

		if association.VirtualGatewayId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-virtual-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *association.VirtualGatewayId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Deleting a virtual gateway will change the state of the association
					In: true,
					// We can't affect the virtual gateway
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectGatewayAssociationAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeDirectConnectGatewayAssociationsInput, *directconnect.DescribeDirectConnectGatewayAssociationsOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeDirectConnectGatewayAssociationsInput, *directconnect.DescribeDirectConnectGatewayAssociationsOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-direct-connect-gateway-association",
		AdapterMetadata: directConnectGatewayAssociationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeDirectConnectGatewayAssociationsInput) (*directconnect.DescribeDirectConnectGatewayAssociationsOutput, error) {
			return client.DescribeDirectConnectGatewayAssociations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*directconnect.DescribeDirectConnectGatewayAssociationsInput, error) {
			// query must be either:
			// - in the format of "directConnectGatewayID/virtualGatewayID"
			// - virtualGatewayID => associatedGatewayID
			dxGatewayID, virtualGatewayID, err := parseDirectConnectGatewayAssociationGetInputQuery(query)
			if err != nil {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: err.Error(),
				}
			}

			if dxGatewayID != "" {
				return &directconnect.DescribeDirectConnectGatewayAssociationsInput{
					DirectConnectGatewayId: &dxGatewayID,
					VirtualGatewayId:       &virtualGatewayID,
				}, nil
			} else {
				return &directconnect.DescribeDirectConnectGatewayAssociationsInput{
					AssociatedGatewayId: &virtualGatewayID,
				}, nil
			}
		},
		InputMapperList: func(scope string) (*directconnect.DescribeDirectConnectGatewayAssociationsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for directconnect-direct-connect-gateway-association, use search",
			}
		},
		OutputMapper: directConnectGatewayAssociationOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *directconnect.Client, scope, query string) (*directconnect.DescribeDirectConnectGatewayAssociationsInput, error) {
			return &directconnect.DescribeDirectConnectGatewayAssociationsInput{
				DirectConnectGatewayId: &query,
			}, nil
		},
	}
}

var directConnectGatewayAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "Direct Connect Gateway Association",
	Type:            "directconnect-direct-connect-gateway-association",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a direct connect gateway association by direct connect gateway ID and virtual gateway ID",
		SearchDescription: "Search direct connect gateway associations by direct connect gateway ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_dx_gateway_association.id"},
	},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks: []string{"directconnect-direct-connect-gateway"},
})

// parseDirectConnectGatewayAssociationGetInputQuery expects a query:
//   - in the format of "directConnectGatewayID/virtualGatewayID"
//   - virtualGatewayID => associatedGatewayID
//
// First returned item is directConnectGatewayID, second is virtualGatewayID
func parseDirectConnectGatewayAssociationGetInputQuery(query string) (string, string, error) {
	ids := strings.Split(query, "/")
	switch len(ids) {
	case 1:
		return "", ids[0], nil
	case 2:
		return ids[0], ids[1], nil
	default:
		return "", "", fmt.Errorf("invalid query, expected in the format of %s or %s, got: %s", directConnectGatewayIDVirtualGatewayIDFormat, virtualGatewayIDFormat, query)
	}
}
