package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// AddressInputMapperGet Maps adapter calls to the correct input for the AZ API
func addressInputMapperGet(scope, query string) (*ec2.DescribeAddressesInput, error) {
	return &ec2.DescribeAddressesInput{
		PublicIps: []string{
			query,
		},
	}, nil
}

// AddressInputMapperList Maps adapter calls to the correct input for the AZ API
func addressInputMapperList(scope string) (*ec2.DescribeAddressesInput, error) {
	return &ec2.DescribeAddressesInput{}, nil
}

// AddressOutputMapper Maps API output to items
func addressOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeAddressesInput, output *ec2.DescribeAddressesOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("empty output")
	}

	items := make([]*sdp.Item, 0)
	var err error
	var attrs *sdp.ItemAttributes

	// An EC2-address, along with an IP is an item that inherently links things
	// and therefore should propagate blast radius in both directions on all
	// links
	bp := &sdp.BlastPropagation{
		In:  true,
		Out: true,
	}

	for _, address := range output.Addresses {
		attrs, err = adapterhelpers.ToAttributesWithExclude(address, "tags")

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ec2-address",
			UniqueAttribute: "PublicIp",
			Scope:           scope,
			Attributes:      attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *address.PublicIp,
						Scope:  "global",
					},
					BlastPropagation: bp,
				},
			},
			Tags: ec2TagsToMap(address.Tags),
		}

		if address.InstanceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-instance",
					Method: sdp.QueryMethod_GET,
					Query:  *address.InstanceId,
					Scope:  scope,
				},
				BlastPropagation: bp,
			})
		}

		if address.CarrierIp != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  *address.CarrierIp,
					Scope:  "global",
				},
				BlastPropagation: bp,
			})
		}

		if address.CustomerOwnedIp != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  *address.CustomerOwnedIp,
					Scope:  "global",
				},
				BlastPropagation: bp,
			})
		}

		if address.NetworkInterfaceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-network-interface",
					Method: sdp.QueryMethod_GET,
					Query:  *address.NetworkInterfaceId,
					Scope:  scope,
				},
				BlastPropagation: bp,
			})
		}

		if address.PrivateIpAddress != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  *address.PrivateIpAddress,
					Scope:  "global",
				},
				BlastPropagation: bp,
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

// NewAddressAdapter Creates a new adapter for aws-Address resources
func NewEC2AddressAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeAddressesInput, *ec2.DescribeAddressesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeAddressesInput, *ec2.DescribeAddressesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-address",
		AdapterMetadata: addressAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
			return client.DescribeAddresses(ctx, input)
		},
		InputMapperGet:  addressInputMapperGet,
		InputMapperList: addressInputMapperList,
		OutputMapper:    addressOutputMapper,
	}
}

var addressAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-address",
	DescriptiveName: "EC2 Address",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an EC2 address by Public IP",
		ListDescription:   "List EC2 addresses",
		SearchDescription: "Search for EC2 addresses by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_eip.public_ip"},
		{TerraformQueryMap: "aws_eip_association.public_ip"},
	},
	PotentialLinks: []string{"ec2-instance", "ip", "ec2-network-interface"},
})
