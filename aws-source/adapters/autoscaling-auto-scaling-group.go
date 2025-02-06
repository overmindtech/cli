package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func autoScalingGroupOutputMapper(_ context.Context, _ *autoscaling.Client, scope string, _ *autoscaling.DescribeAutoScalingGroupsInput, output *autoscaling.DescribeAutoScalingGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	var item sdp.Item
	var attributes *sdp.ItemAttributes
	var err error

	for _, asg := range output.AutoScalingGroups {
		attributes, err = adapterhelpers.ToAttributesWithExclude(asg)

		if err != nil {
			return nil, err
		}

		item = sdp.Item{
			Type:            "autoscaling-auto-scaling-group",
			UniqueAttribute: "AutoScalingGroupName",
			Scope:           scope,
			Attributes:      attributes,
		}

		tags := make(map[string]string)

		for _, tag := range asg.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}

		item.Tags = tags

		if asg.MixedInstancesPolicy != nil {
			if asg.MixedInstancesPolicy.LaunchTemplate != nil {
				if asg.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification != nil {
					if asg.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification.LaunchTemplateId != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ec2-launch-template",
								Method: sdp.QueryMethod_GET,
								Query:  *asg.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification.LaunchTemplateId,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								// Changes to a launch template will affect the ASG
								In: true,
								// Changes to an ASG won't affect the template
								Out: false,
							},
						})
					}
				}
			}
		}

		var a *adapterhelpers.ARN
		var err error

		for _, tgARN := range asg.TargetGroupARNs {
			if a, err = adapterhelpers.ParseARN(tgARN); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "elbv2-target-group",
						Method: sdp.QueryMethod_SEARCH,
						Query:  tgARN,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to a target group won't affect the ASG
						In: false,
						// Changes to an ASG will affect the target group
						Out: true,
					},
				})
			}
		}

		for _, instance := range asg.Instances {
			if instance.InstanceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *instance.InstanceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to an instance could affect the ASG since it
						// could cause it to scale
						In: true,
						// Changes to an ASG can definitely affect an instance
						// since it might be terminated
						Out: true,
					},
				})
			}

			if instance.LaunchTemplate != nil {
				if instance.LaunchTemplate.LaunchTemplateId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-launch-template",
							Method: sdp.QueryMethod_GET,
							Query:  *instance.LaunchTemplate.LaunchTemplateId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to a launch template will affect the ASG
							In: true,
							// Changes to an ASG won't affect the template
							Out: false,
						},
					})
				}
			}
		}

		if asg.ServiceLinkedRoleARN != nil {
			if a, err = adapterhelpers.ParseARN(*asg.ServiceLinkedRoleARN); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-role",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *asg.ServiceLinkedRoleARN,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to a role can affect the functioning of the
						// ASG
						In: true,
						// ASG changes wont affect the role though
						Out: false,
					},
				})
			}
		}

		if asg.LaunchConfigurationName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "autoscaling-launch-configuration",
					Method: sdp.QueryMethod_GET,
					Query:  *asg.LaunchConfigurationName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Very tightly coupled
					In:  true,
					Out: true,
				},
			})
		}

		if asg.LaunchTemplate != nil {
			if asg.LaunchTemplate.LaunchTemplateId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-launch-template",
						Method: sdp.QueryMethod_GET,
						Query:  *asg.LaunchTemplate.LaunchTemplateId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to a launch template will affect the ASG
						In: true,
						// Changes to an ASG won't affect the template
						Out: false,
					},
				})
			}
		}

		if asg.PlacementGroup != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-placement-group",
					Method: sdp.QueryMethod_GET,
					Query:  *asg.PlacementGroup,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to a placement group can affect the ASG
					In: true,
					// Changes to an ASG can affect the placement group
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

//

func NewAutoScalingGroupAdapter(client *autoscaling.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*autoscaling.DescribeAutoScalingGroupsInput, *autoscaling.DescribeAutoScalingGroupsOutput, *autoscaling.Client, *autoscaling.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*autoscaling.DescribeAutoScalingGroupsInput, *autoscaling.DescribeAutoScalingGroupsOutput, *autoscaling.Client, *autoscaling.Options]{
		ItemType:        "autoscaling-auto-scaling-group",
		AccountID:       accountID,
		Region:          region,
		Client:          client,
		AdapterMetadata: autoScalingGroupAdapterMetadata,
		InputMapperGet: func(scope, query string) (*autoscaling.DescribeAutoScalingGroupsInput, error) {
			return &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*autoscaling.DescribeAutoScalingGroupsInput, error) {
			return &autoscaling.DescribeAutoScalingGroupsInput{}, nil
		},
		PaginatorBuilder: func(client *autoscaling.Client, params *autoscaling.DescribeAutoScalingGroupsInput) adapterhelpers.Paginator[*autoscaling.DescribeAutoScalingGroupsOutput, *autoscaling.Options] {
			return autoscaling.NewDescribeAutoScalingGroupsPaginator(client, params)
		},
		DescribeFunc: func(ctx context.Context, client *autoscaling.Client, input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
			return client.DescribeAutoScalingGroups(ctx, input)
		},
		OutputMapper: autoScalingGroupOutputMapper,
	}
}

var autoScalingGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "autoscaling-auto-scaling-group",
	DescriptiveName: "Autoscaling Group",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an Autoscaling Group by name",
		ListDescription:   "List Autoscaling Groups",
		SearchDescription: "Search for Autoscaling Groups by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_autoscaling_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"ec2-launch-template", "elbv2-target-group", "ec2-instance", "iam-role", "autoscaling-launch-configuration", "ec2-placement-group"},
})
