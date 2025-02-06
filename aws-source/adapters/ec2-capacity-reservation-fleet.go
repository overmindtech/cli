package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func capacityReservationFleetOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeCapacityReservationFleetsInput, output *ec2.DescribeCapacityReservationFleetsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, cr := range output.CapacityReservationFleets {
		attributes, err := adapterhelpers.ToAttributesWithExclude(cr, "tags")

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ec2-capacity-reservation-fleet",
			UniqueAttribute: "CapacityReservationFleetId",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            ec2TagsToMap(cr.Tags),
		}

		for _, spec := range cr.InstanceTypeSpecifications {
			if spec.CapacityReservationId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-capacity-reservation",
						Method: sdp.QueryMethod_GET,
						Query:  *spec.CapacityReservationId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the fleet will affect the reservation
						Out: true,
						// The reservation won't affect us
						In: false,
					},
				})
			}
		}

		switch cr.State {
		case types.CapacityReservationFleetStateSubmitted:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.CapacityReservationFleetStateModifying:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.CapacityReservationFleetStateActive:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.CapacityReservationFleetStatePartiallyFulfilled:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.CapacityReservationFleetStateExpiring:
			item.Health = sdp.Health_HEALTH_WARNING.Enum()
		case types.CapacityReservationFleetStateExpired:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		case types.CapacityReservationFleetStateCancelling:
			item.Health = sdp.Health_HEALTH_WARNING.Enum()
		case types.CapacityReservationFleetStateCancelled:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		case types.CapacityReservationFleetStateFailed:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2CapacityReservationFleetAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeCapacityReservationFleetsInput, *ec2.DescribeCapacityReservationFleetsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeCapacityReservationFleetsInput, *ec2.DescribeCapacityReservationFleetsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-capacity-reservation-fleet",
		AdapterMetadata: capacityReservationFleetAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeCapacityReservationFleetsInput) (*ec2.DescribeCapacityReservationFleetsOutput, error) {
			return client.DescribeCapacityReservationFleets(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*ec2.DescribeCapacityReservationFleetsInput, error) {
			return &ec2.DescribeCapacityReservationFleetsInput{
				CapacityReservationFleetIds: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*ec2.DescribeCapacityReservationFleetsInput, error) {
			return &ec2.DescribeCapacityReservationFleetsInput{}, nil
		},
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeCapacityReservationFleetsInput) adapterhelpers.Paginator[*ec2.DescribeCapacityReservationFleetsOutput, *ec2.Options] {
			return ec2.NewDescribeCapacityReservationFleetsPaginator(client, params)
		},
		OutputMapper: capacityReservationFleetOutputMapper,
	}
}

var capacityReservationFleetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-capacity-reservation-fleet",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	DescriptiveName: "Capacity Reservation Fleet",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a capacity reservation fleet by ID",
		ListDescription:   "List capacity reservation fleets",
		SearchDescription: "Search capacity reservation fleets by ARN",
	},
	PotentialLinks: []string{"ec2-capacity-reservation"},
})
