package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func launchTemplateVersionInputMapperGet(scope string, query string) (*ec2.DescribeLaunchTemplateVersionsInput, error) {
	// We are expecting the query to be {id}.{version}
	sections := strings.Split(query, ".")

	if len(sections) != 2 {
		return nil, errors.New("input did not have 2 sections")
	}

	return &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: &sections[0],
		Versions: []string{
			sections[1],
		},
	}, nil
}

func launchTemplateVersionInputMapperList(scope string) (*ec2.DescribeLaunchTemplateVersionsInput, error) {
	return &ec2.DescribeLaunchTemplateVersionsInput{
		Versions: []string{
			"$Latest",
			"$Default",
		},
	}, nil
}

func launchTemplateVersionOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeLaunchTemplateVersionsInput, output *ec2.DescribeLaunchTemplateVersionsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ltv := range output.LaunchTemplateVersions {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(ltv)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		if ltv.LaunchTemplateId != nil && ltv.VersionNumber != nil {
			// Create a custom UAV here since there is no one unique attribute.
			// The new UAV will be {templateId}.{version}
			attrs.Set("VersionIdCombo", fmt.Sprintf("%v.%v", *ltv.LaunchTemplateId, *ltv.VersionNumber))
		} else {
			return nil, errors.New("ec2-launch-template-version must have LaunchTemplateId and VersionNumber populated")
		}

		item := sdp.Item{
			Type:            "ec2-launch-template-version",
			UniqueAttribute: "VersionIdCombo",
			Scope:           scope,
			Attributes:      attrs,
		}

		if lt := ltv.LaunchTemplateData; lt != nil {
			for _, ni := range lt.NetworkInterfaces {
				for _, ip := range ni.Ipv6Addresses {
					if ip.Ipv6Address != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ip",
								Method: sdp.QueryMethod_GET,
								Query:  *ip.Ipv6Address,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								// IPs are always linked
								In:  true,
								Out: true,
							},
						})
					}
				}

				if ni.NetworkInterfaceId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-network-interface",
							Method: sdp.QueryMethod_GET,
							Query:  *ni.NetworkInterfaceId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the network interface will affect the
							// template and vice versa
							In:  true,
							Out: true,
						},
					})
				}

				for _, ip := range ni.PrivateIpAddresses {
					if ip.PrivateIpAddress != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ip",
								Method: sdp.QueryMethod_GET,
								Query:  *ip.PrivateIpAddress,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								// IPs are always linked
								In:  true,
								Out: true,
							},
						})
					}
				}

				if ni.SubnetId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-subnet",
							Method: sdp.QueryMethod_GET,
							Query:  *ni.SubnetId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the subnet will affect the template
							In: true,
							// Changing the template won't affect the subnet
							Out: false,
						},
					})
				}

				for _, group := range ni.Groups {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-security-group",
							Method: sdp.QueryMethod_GET,
							Query:  group,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the security group will affect the
							// template
							In: true,
							// Changing the template won't affect the security
							// group
							Out: false,
						},
					})
				}
			}

			if lt.ImageId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-image",
						Method: sdp.QueryMethod_GET,
						Query:  *lt.ImageId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the image will affect the template
						In: true,
						// Changing the template won't affect the image
						Out: false,
					},
				})
			}

			if lt.KeyName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-key-pair",
						Method: sdp.QueryMethod_GET,
						Query:  *lt.KeyName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the key pair will affect the template
						In: true,
						// Changing the template won't affect the key pair
						Out: false,
					},
				})
			}

			for _, mapping := range lt.BlockDeviceMappings {
				if mapping.Ebs != nil && mapping.Ebs.SnapshotId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-snapshot",
							Method: sdp.QueryMethod_GET,
							Query:  *mapping.Ebs.SnapshotId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the snapshot will affect the template
							In: true,
							// Changing the template won't affect the snapshot
							Out: false,
						},
					})
				}
			}

			if spec := lt.CapacityReservationSpecification; spec != nil {
				if target := spec.CapacityReservationTarget; target != nil {
					if target.CapacityReservationId != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ec2-capacity-reservation",
								Method: sdp.QueryMethod_GET,
								Query:  *target.CapacityReservationId,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								// Changing the capacity reservation will affect
								// the template
								In: true,
								// Changing the template could affect the
								// capacity reservation since it uses it up
								Out: true,
							},
						})
					}
				}
			}

			if lt.Placement != nil {
				if lt.Placement.GroupId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-placement-group",
							Method: sdp.QueryMethod_GET,
							Query:  *lt.Placement.GroupId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the placement group will affect the
							// template
							In: true,
							// Changing the template won't affect the placement
							// group
							Out: false,
						},
					})
				}

				if lt.Placement.HostId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-host",
							Method: sdp.QueryMethod_GET,
							Query:  *lt.Placement.HostId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the host will affect the template
							In: true,
							// Changing the template could affect the host also
							Out: true,
						},
					})
				}
			}

			for _, id := range lt.SecurityGroupIds {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the security group will affect the template
						In: true,
						// Changing the template won't affect the security
						// group
						Out: false,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2LaunchTemplateVersionAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeLaunchTemplateVersionsInput, *ec2.DescribeLaunchTemplateVersionsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeLaunchTemplateVersionsInput, *ec2.DescribeLaunchTemplateVersionsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-launch-template-version",
		AdapterMetadata: launchTemplateVersionAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeLaunchTemplateVersionsInput) (*ec2.DescribeLaunchTemplateVersionsOutput, error) {
			return client.DescribeLaunchTemplateVersions(ctx, input)
		},
		InputMapperGet:  launchTemplateVersionInputMapperGet,
		InputMapperList: launchTemplateVersionInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeLaunchTemplateVersionsInput) adapterhelpers.Paginator[*ec2.DescribeLaunchTemplateVersionsOutput, *ec2.Options] {
			return ec2.NewDescribeLaunchTemplateVersionsPaginator(client, params)
		},
		OutputMapper: launchTemplateVersionOutputMapper,
	}
}

var launchTemplateVersionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-launch-template-version",
	DescriptiveName: "Launch Template Version",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a launch template version by {templateId}.{version}",
		ListDescription:   "List all launch template versions",
		SearchDescription: "Search launch template versions by ARN",
	},
	PotentialLinks: []string{"ec2-network-interface", "ec2-subnet", "ec2-security-group", "ec2-image", "ec2-key-pair", "ec2-snapshot", "ec2-capacity-reservation", "ec2-placement-group", "ec2-host", "ip"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
