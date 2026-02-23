package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

var (
	codePending      = int32(0)
	codeRunning      = int32(16)
	codeShuttingDown = int32(32)
	codeTerminated   = int32(48)
	codeStopping     = int32(64)
	codeStopped      = int32(80)
)

func instanceInputMapperGet(scope, query string) (*ec2.DescribeInstancesInput, error) {
	return &ec2.DescribeInstancesInput{
		InstanceIds: []string{
			query,
		},
	}, nil
}

func instanceInputMapperList(scope string) (*ec2.DescribeInstancesInput, error) {
	return &ec2.DescribeInstancesInput{}, nil
}

func instanceOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeInstancesInput, output *ec2.DescribeInstancesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			attrs, err := ToAttributesWithExclude(instance, "tags")

			if err != nil {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_OTHER,
					ErrorString: err.Error(),
					Scope:       scope,
				}
			}

			item := sdp.Item{
				Type:            "ec2-instance",
				UniqueAttribute: "InstanceId",
				Scope:           scope,
				Attributes:      attrs,
				Tags:            ec2TagsToMap(instance.Tags),
				LinkedItemQueries: []*sdp.LinkedItemQuery{
					{
						Query: &sdp.Query{
							// Always get the status
							Type:   "ec2-instance-status",
							Method: sdp.QueryMethod_GET,
							Query:  *instance.InstanceId,
							Scope:  scope,
						},
					},
					{
						Query: &sdp.Query{
							// Get CloudWatch metrics for this instance
							Type:   "cloudwatch-instance-metric",
							Method: sdp.QueryMethod_GET,
							Query:  *instance.InstanceId,
							Scope:  scope,
						},
					},
				},
			}

			if instance.State != nil {
				switch aws.ToInt32(instance.State.Code) {
				case codeRunning:
					item.Health = sdp.Health_HEALTH_OK.Enum()
				case codePending:
					item.Health = sdp.Health_HEALTH_PENDING.Enum()
				case codeShuttingDown:
					item.Health = sdp.Health_HEALTH_PENDING.Enum()
				case codeStopping:
					item.Health = sdp.Health_HEALTH_PENDING.Enum()
				case codeTerminated, codeStopped:
					// No health for things that aren't running
				}
			}

			if instance.IamInstanceProfile != nil {
				// Prefer the ARN
				if instance.IamInstanceProfile.Arn != nil {
					if arn, err := ParseARN(*instance.IamInstanceProfile.Arn); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "iam-instance-profile",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *instance.IamInstanceProfile.Arn,
								Scope:  FormatScope(arn.AccountID, arn.Region),
							},
						})
					}
				} else if instance.IamInstanceProfile.Id != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "iam-instance-profile",
							Method: sdp.QueryMethod_GET,
							Query:  *instance.IamInstanceProfile.Id,
							Scope:  scope,
						},
					})
				}
			}

			if instance.CapacityReservationId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-capacity-reservation",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.CapacityReservationId,
						Scope:  scope,
					},
				})
			}

			for _, assoc := range instance.ElasticGpuAssociations {
				if assoc.ElasticGpuId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-elastic-gpu",
							Method: sdp.QueryMethod_GET,
							Query:  *assoc.ElasticGpuId,
							Scope:  scope,
						},
					})
				}
			}

			for _, assoc := range instance.ElasticInferenceAcceleratorAssociations {
				if assoc.ElasticInferenceAcceleratorArn != nil {
					if arn, err := ParseARN(*assoc.ElasticInferenceAcceleratorArn); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "elastic-inference-accelerator",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *assoc.ElasticInferenceAcceleratorArn,
								Scope:  FormatScope(arn.AccountID, arn.Region),
							},
						})
					}
				}
			}

			for _, license := range instance.Licenses {
				if license.LicenseConfigurationArn != nil {
					if arn, err := ParseARN(*license.LicenseConfigurationArn); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "license-manager-license-configuration",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *license.LicenseConfigurationArn,
								Scope:  FormatScope(arn.AccountID, arn.Region),
							},
						})
					}
				}
			}

			if instance.OutpostArn != nil {
				if arn, err := ParseARN(*instance.OutpostArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "outposts-outpost",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *instance.OutpostArn,
							Scope:  FormatScope(arn.AccountID, arn.Region),
						},
					})
				}
			}

			if instance.SpotInstanceRequestId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-spot-instance-request",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.SpotInstanceRequestId,
						Scope:  scope,
					},
				})
			}

			if instance.ImageId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-image",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.ImageId,
						Scope:  scope,
					},
				})
			}

			if instance.KeyName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-key-pair",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.KeyName,
						Scope:  scope,
					},
				})
			}

			if instance.Placement != nil {
				if instance.Placement.GroupId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-placement-group",
							Method: sdp.QueryMethod_GET,
							Query:  *instance.Placement.GroupId,
							Scope:  scope,
						},
					})
				}
			}

			if instance.Ipv6Address != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.Ipv6Address,
						Scope:  "global",
					},
				})
			}

			for _, nic := range instance.NetworkInterfaces {
				// IPs
				for _, ip := range nic.Ipv6Addresses {
					if ip.Ipv6Address != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ip",
								Method: sdp.QueryMethod_GET,
								Query:  *ip.Ipv6Address,
								Scope:  "global",
							},
						})
					}
				}

				for _, ip := range nic.PrivateIpAddresses {
					if ip.PrivateIpAddress != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ip",
								Method: sdp.QueryMethod_GET,
								Query:  *ip.PrivateIpAddress,
								Scope:  "global",
							},
						})
					}
				}

				// Subnet
				if nic.SubnetId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-subnet",
							Method: sdp.QueryMethod_GET,
							Query:  *nic.SubnetId,
							Scope:  scope,
						},
					})
				}

				// VPC
				if nic.VpcId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-vpc",
							Method: sdp.QueryMethod_GET,
							Query:  *nic.VpcId,
							Scope:  scope,
						},
					})
				}
			}

			if instance.PublicDnsName != nil && *instance.PublicDnsName != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *instance.PublicDnsName,
						Scope:  "global",
					},
				})
			}

			if instance.PublicIpAddress != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.PublicIpAddress,
						Scope:  "global",
					},
				})
			}

			// Security groups
			for _, group := range instance.SecurityGroups {
				if group.GroupId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-security-group",
							Method: sdp.QueryMethod_GET,
							Query:  *group.GroupId,
							Scope:  scope,
						},
					})
				}
			}

			for _, mapping := range instance.BlockDeviceMappings {
				if mapping.Ebs != nil && mapping.Ebs.VolumeId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-volume",
							Method: sdp.QueryMethod_GET,
							Query:  *mapping.Ebs.VolumeId,
							Scope:  scope,
						},
					})
				}
			}

			items = append(items, &item)
		}
	}

	return items, nil
}

func NewEC2InstanceAdapter(client *ec2.Client, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*ec2.DescribeInstancesInput, *ec2.DescribeInstancesOutput, *ec2.Client, *ec2.Options] {
	return &DescribeOnlyAdapter[*ec2.DescribeInstancesInput, *ec2.DescribeInstancesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-instance",
		AdapterMetadata: ec2InstanceAdapterMetadata,
		cache:        cache,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return client.DescribeInstances(ctx, input)
		},
		InputMapperGet:  instanceInputMapperGet,
		InputMapperList: instanceInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeInstancesInput) Paginator[*ec2.DescribeInstancesOutput, *ec2.Options] {
			return ec2.NewDescribeInstancesPaginator(client, params)
		},
		OutputMapper: instanceOutputMapper,
	}
}

var ec2InstanceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-instance",
	DescriptiveName: "EC2 Instance",
	PotentialLinks:  []string{"ec2-instance-status", "cloudwatch-instance-metric", "iam-instance-profile", "ec2-capacity-reservation", "ec2-elastic-gpu", "elastic-inference-accelerator", "license-manager-license-configuration", "outposts-outpost", "ec2-spot-instance-request", "ec2-image", "ec2-key-pair", "ec2-placement-group", "ip", "ec2-subnet", "ec2-vpc", "dns", "ec2-security-group", "ec2-volume"},
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an EC2 instance by ID",
		ListDescription:   "List all EC2 instances",
		SearchDescription: "Search EC2 instances by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "aws_instance.id",
		},
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "aws_instance.arn",
		},
	},
})
