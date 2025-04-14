package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func resourceRecordSetGetFunc(ctx context.Context, client *route53.Client, scope, query string) (*types.ResourceRecordSet, error) {
	return nil, errors.New("get is not supported for route53-resource-record-set. Use search")
}

// ResourceRecordSetSearchFunc Search func that accepts a hosted zone or a
// terraform ID in the format {hostedZone}_{recordName}_{type}. Unfortunately
// the "name" means the record name within the scope of the hosted zone, not the
// full FQDN. This is something that Terraform does to match the AWS GUI, where
// you specify a name like "foo" and then you end up with a record like
// "foo.example.com.". That record has a "name" attribute, but it's set to
// "foo.example.com.".
//
// Because of this behaviour we need to construct the full name, rather than
// just the half-name. You can see that the terraform provider itself also does
// this in `findResourceRecordSetByFourPartKey`:
// https://github.com/hashicorp/terraform-provider-aws/blob/main/internal/service/route53/record.go#L786-L825
func resourceRecordSetSearchFunc(ctx context.Context, client *route53.Client, scope, query string) ([]*types.ResourceRecordSet, error) {
	splits := strings.Split(query, "_")

	var out *route53.ListResourceRecordSetsOutput
	var err error
	if len(splits) == 3 {
		hostedZoneID := splits[0]
		recordName := splits[1]
		recordType := splits[2]

		var zoneResp *route53.GetHostedZoneOutput
		// In this case we have a terraform ID. We have to get the details of the hosted zone first
		zoneResp, err = client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
			Id: &hostedZoneID,
		})
		if err != nil {
			return nil, err
		}
		if zoneResp.HostedZone == nil {
			return nil, fmt.Errorf("hosted zone %s not found", hostedZoneID)
		}

		// If the name is the same as the FQDN of the hosted zone, we don't have
		// to append it otherwise it'll be in there twice. It seems that NS and
		// MX records sometimes have the full FQDN in the name
		zoneFQDN := strings.TrimSuffix(*zoneResp.HostedZone.Name, ".")
		var fullName string
		if recordName == zoneFQDN {
			fullName = recordName
		} else {
			// Calculate the full FQDN based on the hosted zone name and the record name
			fullName = recordName + "." + *zoneResp.HostedZone.Name
		}

		var maxItems int32 = 1
		req := route53.ListResourceRecordSetsInput{
			HostedZoneId:    &hostedZoneID,
			StartRecordName: &fullName,
			StartRecordType: types.RRType(recordType),
			MaxItems:        &maxItems,
		}
		out, err = client.ListResourceRecordSets(ctx, &req)
	} else {
		// In this case we have a hosted zone ID
		out, err = client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
			HostedZoneId: &query,
		})
	}

	if err != nil {
		return nil, err
	}

	records := make([]*types.ResourceRecordSet, 0, len(out.ResourceRecordSets))

	for _, record := range out.ResourceRecordSets {
		records = append(records, &record)
	}

	return records, nil
}

func resourceRecordSetItemMapper(_, scope string, awsItem *types.ResourceRecordSet) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "route53-resource-record-set",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
	}

	if awsItem.AliasTarget != nil {
		if awsItem.AliasTarget.DNSName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *awsItem.AliasTarget.DNSName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS aliases links
					In:  true,
					Out: true,
				},
			})
		}
	}

	for _, record := range awsItem.ResourceRecords {
		if record.Value != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *record.Value,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS aliases links
					In:  true,
					Out: true,
				},
			})
		}
	}

	if awsItem.HealthCheckId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "route53-health-check",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.HealthCheckId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Health check links tightly
				In:  true,
				Out: true,
			},
		})
	}

	return &item, nil
}

func NewRoute53ResourceRecordSetAdapter(client *route53.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*types.ResourceRecordSet, *route53.Client, *route53.Options] {
	return &adapterhelpers.GetListAdapter[*types.ResourceRecordSet, *route53.Client, *route53.Options]{
		ItemType:        "route53-resource-record-set",
		Client:          client,
		DisableList:     true,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         resourceRecordSetGetFunc,
		ItemMapper:      resourceRecordSetItemMapper,
		SearchFunc:      resourceRecordSetSearchFunc,
		AdapterMetadata: resourceRecordSetAdapterMetadata,
	}
}

var resourceRecordSetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "route53-resource-record-set",
	DescriptiveName: "Route53 Record Set",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Route53 record Set by name",
		SearchDescription: "Search for a record set by hosted zone ID in the format \"/hostedzone/JJN928734JH7HV\" or \"JJN928734JH7HV\" or by terraform ID in the format \"{hostedZone}_{recordName}_{type}\"",
	},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks: []string{"dns", "route53-health-check"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_route53_record.arn", TerraformMethod: sdp.QueryMethod_SEARCH},
		{TerraformQueryMap: "aws_route53_record.id", TerraformMethod: sdp.QueryMethod_SEARCH},
	},
})
