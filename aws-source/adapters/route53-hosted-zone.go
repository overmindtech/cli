package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func hostedZoneGetFunc(ctx context.Context, client *route53.Client, scope, query string) (*types.HostedZone, error) {
	out, err := client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: &query,
	})

	if err != nil {
		return nil, err
	}

	return out.HostedZone, nil
}

func hostedZoneListFunc(ctx context.Context, client *route53.Client, scope string) ([]*types.HostedZone, error) {
	out, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})

	if err != nil {
		return nil, err
	}

	zones := make([]*types.HostedZone, 0, len(out.HostedZones))

	for _, zone := range out.HostedZones {
		zones = append(zones, &zone)
	}

	return zones, nil
}

func hostedZoneItemMapper(_, scope string, awsItem *types.HostedZone) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "route53-hosted-zone",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "route53-resource-record-set",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *awsItem.Id,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the hosted zone can affect the resource record set
					Out: true,
					// The resource record set won't affect the hosted zone
					In: false,
				},
			},
		},
	}

	return &item, nil
}

func NewRoute53HostedZoneAdapter(client *route53.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.HostedZone, *route53.Client, *route53.Options] {
	return &adapterhelpers.GetListAdapter[*types.HostedZone, *route53.Client, *route53.Options]{
		ItemType:        "route53-hosted-zone",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         hostedZoneGetFunc,
		ListFunc:        hostedZoneListFunc,
		ItemMapper:      hostedZoneItemMapper,
		AdapterMetadata: hostedZoneAdapterMetadata,
		ListTagsFunc: func(ctx context.Context, hz *types.HostedZone, c *route53.Client) (map[string]string, error) {
			if hz.Id == nil {
				return nil, nil
			}

			// Strip the initial prefix
			id := strings.TrimPrefix(*hz.Id, "/hostedzone/")

			out, err := c.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
				ResourceId:   &id,
				ResourceType: types.TagResourceTypeHostedzone,
			})

			if err != nil {
				return nil, err
			}

			return route53TagsToMap(out.ResourceTagSet.Tags), nil
		},
	}
}

var hostedZoneAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "route53-hosted-zone",
	DescriptiveName: "Hosted Zone",
	PotentialLinks:  []string{"route53-resource-record-set"},
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a hosted zone by ID",
		ListDescription:   "List all hosted zones",
		SearchDescription: "Search for a hosted zone by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_route53_hosted_zone_dnssec.id"},
		{TerraformQueryMap: "aws_route53_zone.zone_id"},
		{TerraformQueryMap: "aws_route53_zone_association.zone_id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
