package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func capacityReservationOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeCapacityReservationsInput, output *ec2.DescribeCapacityReservationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, cr := range output.CapacityReservations {
		attributes, err := adapterhelpers.ToAttributesWithExclude(cr, "tags")

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ec2-capacity-reservation",
			UniqueAttribute: "CapacityReservationId",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            ec2TagsToMap(cr.Tags),
		}

		if cr.CapacityReservationFleetId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-capacity-reservation-fleet",
					Method: sdp.QueryMethod_GET,
					Query:  *cr.CapacityReservationFleetId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the fleet will affect this
					In: true,
					// We can't affect the fleet
					Out: false,
				},
			})
		}

		if cr.OutpostArn != nil {
			if arn, err := adapterhelpers.ParseARN(*cr.OutpostArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "outposts-outpost",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cr.OutpostArn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the outpost will affect this
						In: true,
						// We can't affect the outpost
						Out: false,
					},
				})
			}
		}

		if cr.PlacementGroupArn != nil {
			if arn, err := adapterhelpers.ParseARN(*cr.PlacementGroupArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-placement-group",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cr.PlacementGroupArn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the placement group will affect this
						In: true,
						// We can't affect the placement group
						Out: false,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2CapacityReservationAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeCapacityReservationsInput, *ec2.DescribeCapacityReservationsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeCapacityReservationsInput, *ec2.DescribeCapacityReservationsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-capacity-reservation",
		AdapterMetadata: capacityReservationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeCapacityReservationsInput) (*ec2.DescribeCapacityReservationsOutput, error) {
			return client.DescribeCapacityReservations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*ec2.DescribeCapacityReservationsInput, error) {
			return &ec2.DescribeCapacityReservationsInput{
				CapacityReservationIds: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*ec2.DescribeCapacityReservationsInput, error) {
			return &ec2.DescribeCapacityReservationsInput{}, nil
		},
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeCapacityReservationsInput) adapterhelpers.Paginator[*ec2.DescribeCapacityReservationsOutput, *ec2.Options] {
			return ec2.NewDescribeCapacityReservationsPaginator(client, params)
		},
		OutputMapper: capacityReservationOutputMapper,
	}
}

var capacityReservationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-capacity-reservation",
	DescriptiveName: "Capacity Reservation",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a capacity reservation fleet by ID",
		ListDescription:   "List capacity reservation fleets",
		SearchDescription: "Search capacity reservation fleets by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ec2_capacity_reservation_fleet.id"},
	},
	PotentialLinks: []string{"outposts-outpost", "ec2-placement-group", "ec2-capacity-reservation-fleet"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
