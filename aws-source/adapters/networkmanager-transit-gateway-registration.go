package adapters

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func transitGatewayRegistrationOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetTransitGatewayRegistrationsInput, output *networkmanager.GetTransitGatewayRegistrationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, r := range output.TransitGatewayRegistrations {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(r)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		if r.GlobalNetworkId == nil || r.TransitGatewayArn == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId or transitGatewayArn is nil for transit gateway registration"))
		}

		attrs.Set("GlobalNetworkIdWithTransitGatewayARN", idWithGlobalNetwork(*r.GlobalNetworkId, *r.TransitGatewayArn))

		item := sdp.Item{
			Type:            "networkmanager-transit-gateway-registration",
			UniqueAttribute: "GlobalNetworkIdWithTransitGatewayARN",
			Scope:           scope,
			Attributes:      attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "networkmanager-global-network",
						Method: sdp.QueryMethod_GET,
						Query:  *r.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		// ARN example: "arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234"
		if r.TransitGatewayArn != nil {
			if arn, err := adapterhelpers.ParseARN(*r.TransitGatewayArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-transit-gateway",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *r.TransitGatewayArn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerTransitGatewayRegistrationAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetTransitGatewayRegistrationsInput, *networkmanager.GetTransitGatewayRegistrationsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetTransitGatewayRegistrationsInput, *networkmanager.GetTransitGatewayRegistrationsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-transit-gateway-registration",
		AdapterMetadata: transitGatewayRegistrationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetTransitGatewayRegistrationsInput) (*networkmanager.GetTransitGatewayRegistrationsOutput, error) {
			return client.GetTransitGatewayRegistrations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetTransitGatewayRegistrationsInput, error) {
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, sdp.NewQueryError(errors.New("invalid query for networkmanager-transit-gateway-registration get function, must be in the format {globalNetworkId}|{transitGatewayARN}"))
			}

			// we are using a custom id of {globalNetworkId}|{transitGatewayARN}
			// e.g. searching from ec2-transit-gateway
			return &networkmanager.GetTransitGatewayRegistrationsInput{
				GlobalNetworkId: &sections[0],
				TransitGatewayArns: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetTransitGatewayRegistrationsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-transit-gateway-registration, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetTransitGatewayRegistrationsInput) adapterhelpers.Paginator[*networkmanager.GetTransitGatewayRegistrationsOutput, *networkmanager.Options] {
			return networkmanager.NewGetTransitGatewayRegistrationsPaginator(client, params)
		},
		OutputMapper: transitGatewayRegistrationOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetTransitGatewayRegistrationsInput, error) {
			// Search by GlobalNetworkId
			return &networkmanager.GetTransitGatewayRegistrationsInput{
				GlobalNetworkId: &query,
			}, nil
		},
	}
}

var transitGatewayRegistrationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-transit-gateway-registration",
	DescriptiveName: "Networkmanager Transit Gateway Registrations",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Transit Gateway Registrations",
		ListDescription:   "List all Networkmanager Transit Gateway Registrations",
		SearchDescription: "Search for Networkmanager Transit Gateway Registrations by GlobalNetworkId",
	},
	PotentialLinks: []string{"networkmanager-global-network", "ec2-transit-gateway"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
