package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func scalingPolicyOutputMapper(_ context.Context, _ *autoscaling.Client, scope string, _ *autoscaling.DescribePoliciesInput, output *autoscaling.DescribePoliciesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0, len(output.ScalingPolicies))

	for _, policy := range output.ScalingPolicies {
		// Both AutoScalingGroupName and PolicyName are required to form a unique identifier
		if policy.AutoScalingGroupName == nil || policy.PolicyName == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "policy is missing AutoScalingGroupName or PolicyName",
			}
		}

		attributes, err := adapterhelpers.ToAttributesWithExclude(policy)

		if err != nil {
			return nil, err
		}

		// The uniqueAttributeValue is the combination of ASG name and policy name
		// i.e., "my-asg/scale-up-policy"
		err = attributes.Set("UniqueName", fmt.Sprintf("%s/%s", *policy.AutoScalingGroupName, *policy.PolicyName))
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "autoscaling-auto-scaling-policy",
			UniqueAttribute: "UniqueName",
			Scope:           scope,
			Attributes:      attributes,
		}

		// Link to the Auto Scaling Group (already validated as non-nil above)
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "autoscaling-auto-scaling-group",
				Method: sdp.QueryMethod_GET,
				Query:  *policy.AutoScalingGroupName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the ASG can affect the policy (e.g., if the
				// ASG is deleted, the policy is also deleted)
				In: true,
				// Changes to the policy can affect the ASG behavior
				Out: true,
			},
		})

		// Link to CloudWatch Alarms
		for _, alarm := range policy.Alarms {
			if alarm.AlarmName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudwatch-alarm",
						Method: sdp.QueryMethod_GET,
						Query:  *alarm.AlarmName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Alarms trigger policies, so changes to alarms can
						// affect policy execution
						In: true,
						// Changes to the policy don't affect the alarm itself
						Out: false,
					},
				})
			}
		}

		// Link to ELBv2 resources from TargetTrackingConfiguration
		if policy.TargetTrackingConfiguration != nil &&
			policy.TargetTrackingConfiguration.PredefinedMetricSpecification != nil &&
			policy.TargetTrackingConfiguration.PredefinedMetricSpecification.ResourceLabel != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries,
				parseResourceLabelLinks(*policy.TargetTrackingConfiguration.PredefinedMetricSpecification.ResourceLabel, scope)...)
		}

		// Link to ELBv2 resources from PredictiveScalingConfiguration
		if policy.PredictiveScalingConfiguration != nil {
			for _, metricSpec := range policy.PredictiveScalingConfiguration.MetricSpecifications {
				// PredefinedMetricPairSpecification
				if metricSpec.PredefinedMetricPairSpecification != nil &&
					metricSpec.PredefinedMetricPairSpecification.ResourceLabel != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries,
						parseResourceLabelLinks(*metricSpec.PredefinedMetricPairSpecification.ResourceLabel, scope)...)
				}
				// PredefinedLoadMetricSpecification
				if metricSpec.PredefinedLoadMetricSpecification != nil &&
					metricSpec.PredefinedLoadMetricSpecification.ResourceLabel != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries,
						parseResourceLabelLinks(*metricSpec.PredefinedLoadMetricSpecification.ResourceLabel, scope)...)
				}
				// PredefinedScalingMetricSpecification
				if metricSpec.PredefinedScalingMetricSpecification != nil &&
					metricSpec.PredefinedScalingMetricSpecification.ResourceLabel != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries,
						parseResourceLabelLinks(*metricSpec.PredefinedScalingMetricSpecification.ResourceLabel, scope)...)
				}
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

// parseResourceLabelLinks parses a ResourceLabel string and returns LinkedItemQueries
// for ELBv2 target groups and load balancers.
// The ResourceLabel format is: app/my-alb/778d41231b141a0f/targetgroup/my-alb-target-group/943f017f100becff
// Where:
//   - app/<lb-name>/<hash> is the final portion of an Application Load Balancer ARN
//   - net/<lb-name>/<hash> is the final portion of a Network Load Balancer ARN
//   - gwy/<lb-name>/<hash> is the final portion of a Gateway Load Balancer ARN
//   - targetgroup/<tg-name>/<hash> is the final portion of the target group ARN
func parseResourceLabelLinks(resourceLabel string, scope string) []*sdp.LinkedItemQuery {
	var links []*sdp.LinkedItemQuery

	sections := strings.Split(resourceLabel, "/")
	// Expected format: {app|net|gwy}/lb-name/hash/targetgroup/tg-name/hash (6 sections)
	if len(sections) < 6 {
		return links
	}

	// Extract load balancer name (index 1 when starting with "app", "net", or "gwy")
	// These prefixes correspond to ALB, NLB, and GLB respectively
	if (sections[0] == "app" || sections[0] == "net" || sections[0] == "gwy") && sections[1] != "" {
		links = append(links, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "elbv2-load-balancer",
				Method: sdp.QueryMethod_GET,
				Query:  sections[1],
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the load balancer can affect the scaling policy
				// (e.g., if the LB is deleted or modified)
				In: true,
				// The scaling policy doesn't directly affect the load balancer
				Out: false,
			},
		})
	}

	// Find "targetgroup" and extract the target group name (next element)
	for i, section := range sections {
		if section == "targetgroup" && i+1 < len(sections) && sections[i+1] != "" {
			links = append(links, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-target-group",
					Method: sdp.QueryMethod_GET,
					Query:  sections[i+1],
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the target group can affect the scaling policy
					// (e.g., request count metrics come from the target group)
					In: true,
					// The scaling policy doesn't directly affect the target group
					Out: false,
				},
			})
			break
		}
	}

	return links
}

func NewAutoScalingPolicyAdapter(client *autoscaling.Client, accountID string, region string, cache sdpcache.Cache) *adapterhelpers.DescribeOnlyAdapter[*autoscaling.DescribePoliciesInput, *autoscaling.DescribePoliciesOutput, *autoscaling.Client, *autoscaling.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*autoscaling.DescribePoliciesInput, *autoscaling.DescribePoliciesOutput, *autoscaling.Client, *autoscaling.Options]{
		ItemType:        "autoscaling-auto-scaling-policy",
		AccountID:       accountID,
		Region:          region,
		Client:          client,
		AdapterMetadata: scalingPolicyAdapterMetadata,
		SDPCache:        cache,
		InputMapperGet: func(scope, query string) (*autoscaling.DescribePoliciesInput, error) {
			// Query must be in the format: asgName/policyName
			// e.g., "my-asg/scale-up-policy"
			parts := strings.SplitN(query, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format asgName/policyName, got: %s", query),
				}
			}

			return &autoscaling.DescribePoliciesInput{
				AutoScalingGroupName: &parts[0],
				PolicyNames:          []string{parts[1]},
			}, nil
		},
		InputMapperList: func(scope string) (*autoscaling.DescribePoliciesInput, error) {
			return &autoscaling.DescribePoliciesInput{}, nil
		},
		InputMapperSearch: func(ctx context.Context, client *autoscaling.Client, scope, query string) (*autoscaling.DescribePoliciesInput, error) {
			// Parse the ARN to extract the policy name and ASG name
			// Scaling Policy ARNs have the format:
			// arn:aws:autoscaling:region:account-id:scalingPolicy:uuid:autoScalingGroupName/group-name:policyName/policy-name
			arn, err := adapterhelpers.ParseARN(query)
			if err != nil {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid ARN format for autoscaling-auto-scaling-policy",
				}
			}

			// Check if it's an autoscaling ARN
			if arn.Service != "autoscaling" {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "ARN is not for autoscaling service",
				}
			}

			// The resource part looks like: scalingPolicy:uuid:autoScalingGroupName/group-name:policyName/policy-name
			// We need to extract the ASG name and policy name
			if !strings.Contains(arn.Resource, "scalingPolicy:") {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "ARN is not for a scaling policy",
				}
			}

			var asgName, policyName string

			// Extract ASG name
			if strings.Contains(arn.Resource, "autoScalingGroupName/") {
				parts := strings.Split(arn.Resource, "autoScalingGroupName/")
				if len(parts) >= 2 {
					// Now we have something like "group-name:policyName/policy-name"
					asgPart := parts[1]
					// Split on ":policyName/" to separate ASG name from policy name part
					if strings.Contains(asgPart, ":policyName/") {
						asgPolicyParts := strings.Split(asgPart, ":policyName/")
						if len(asgPolicyParts) == 2 {
							asgName = asgPolicyParts[0]
							policyName = asgPolicyParts[1]
						}
					}
				}
			}

			if asgName == "" || policyName == "" {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "could not extract ASG name and policy name from ARN",
				}
			}

			return &autoscaling.DescribePoliciesInput{
				AutoScalingGroupName: &asgName,
				PolicyNames:          []string{policyName},
			}, nil
		},
		PaginatorBuilder: func(client *autoscaling.Client, params *autoscaling.DescribePoliciesInput) adapterhelpers.Paginator[*autoscaling.DescribePoliciesOutput, *autoscaling.Options] {
			return autoscaling.NewDescribePoliciesPaginator(client, params)
		},
		DescribeFunc: func(ctx context.Context, client *autoscaling.Client, input *autoscaling.DescribePoliciesInput) (*autoscaling.DescribePoliciesOutput, error) {
			return client.DescribePolicies(ctx, input)
		},
		OutputMapper: scalingPolicyOutputMapper,
	}
}

var scalingPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "autoscaling-auto-scaling-policy",
	DescriptiveName: "Autoscaling Policy",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an Autoscaling Policy by {asgName}/{policyName}",
		ListDescription:   "List Autoscaling Policies",
		SearchDescription: "Search for Autoscaling Policies by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_autoscaling_policy.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"autoscaling-auto-scaling-group", "cloudwatch-alarm", "elbv2-load-balancer", "elbv2-target-group"},
})
