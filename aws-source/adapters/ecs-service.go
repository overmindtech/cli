package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// ServiceIncludeFields Fields that we want included by default
var ServiceIncludeFields = []types.ServiceField{
	types.ServiceFieldTags,
}

func serviceGetFunc(ctx context.Context, client ECSClient, scope string, input *ecs.DescribeServicesInput) (*sdp.Item, error) {
	if input == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no input provided",
		}
	}
	out, err := client.DescribeServices(ctx, input)

	if err != nil {
		return nil, err
	}

	if len(out.Services) != 1 {
		return nil, fmt.Errorf("got %v Services, expected 1", len(out.Services))
	}

	service := out.Services[0]

	// Before we convert to attributes we want to extract the task sets to link
	// to and then delete the info. This because the response embeds the entire
	// task set which is unnecessary since it'll be returned by ecs-task-set
	taskSetIds := make([]string, 0)

	for _, ts := range service.TaskSets {
		if ts.Id != nil {
			taskSetIds = append(taskSetIds, *ts.Id)
		}
	}

	service.TaskSets = []types.TaskSet{}

	attributes, err := adapterhelpers.ToAttributesWithExclude(service, "tags")

	if err != nil {
		return nil, err
	}

	if service.ServiceArn != nil {
		if a, err := adapterhelpers.ParseARN(*service.ServiceArn); err == nil {
			attributes.Set("ServiceFullName", a.Resource)
		}
	}

	item := sdp.Item{
		Type:            "ecs-service",
		UniqueAttribute: "ServiceFullName",
		Scope:           scope,
		Attributes:      attributes,
		Tags:            ecsTagsToMap(service.Tags),
	}

	if service.Status != nil {
		switch *service.Status {
		case "ACTIVE":
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case "DRAINING":
			item.Health = sdp.Health_HEALTH_WARNING.Enum()
		case "INACTIVE":
			item.Health = nil
		}
	}

	var a *adapterhelpers.ARN

	if service.ClusterArn != nil {
		if a, err = adapterhelpers.ParseARN(*service.ClusterArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ecs-cluster",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *service.ClusterArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the cluster will affect the service
					In: true,
					// The service should be able to affect the cluster
					Out: false,
				},
			})
		}
	}

	for _, lb := range service.LoadBalancers {
		if lb.TargetGroupArn != nil {
			if a, err = adapterhelpers.ParseARN(*lb.TargetGroupArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "elbv2-target-group",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *lb.TargetGroupArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// These are tightly linked
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	for _, sr := range service.ServiceRegistries {
		if sr.RegistryArn != nil {
			if a, err = adapterhelpers.ParseARN(*sr.RegistryArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "servicediscovery-service",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *sr.RegistryArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// These are tightly linked
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	if service.TaskDefinition != nil {
		if a, err = adapterhelpers.ParseARN(*service.TaskDefinition); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ecs-task-definition",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *service.TaskDefinition,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the task definition will affect the service
					In: true,
					// The service shouldn't affect the task definition itself
					Out: false,
				},
			})
		}
	}

	for _, deployment := range service.Deployments {
		if deployment.TaskDefinition != nil {
			if a, err = adapterhelpers.ParseARN(*deployment.TaskDefinition); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ecs-task-definition",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *deployment.TaskDefinition,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the task definition will affect the service
						In: true,
						// The service shouldn't affect the task definition itself
						Out: false,
					},
				})
			}
		}

		for _, strategy := range deployment.CapacityProviderStrategy {
			if strategy.CapacityProvider != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ecs-capacity-provider",
						Method: sdp.QueryMethod_GET,
						Query:  *strategy.CapacityProvider,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the capacity provider will affect the service
						In: true,
						// The service shouldn't affect the capacity provider itself
						Out: false,
					},
				})
			}
		}

		if deployment.NetworkConfiguration != nil {
			if deployment.NetworkConfiguration.AwsvpcConfiguration != nil {
				for _, subnet := range deployment.NetworkConfiguration.AwsvpcConfiguration.Subnets {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-subnet",
							Method: sdp.QueryMethod_GET,
							Query:  subnet,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the subnet will affect the service
							In: true,
							// The service shouldn't affect the subnet
							Out: false,
						},
					})
				}

				for _, sg := range deployment.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ecs-security-group",
							Method: sdp.QueryMethod_GET,
							Query:  sg,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the security group will affect the service
							In: true,
							// The service shouldn't affect the security group
							Out: false,
						},
					})
				}
			}
		}

		if deployment.ServiceConnectConfiguration != nil {
			for _, svc := range deployment.ServiceConnectConfiguration.Services {
				for _, alias := range svc.ClientAliases {
					if alias.DnsName != nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "dns",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *alias.DnsName,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								// DNS always links
								In:  true,
								Out: true,
							},
						})
					}
				}
			}
		}

		for _, cr := range deployment.ServiceConnectResources {
			if cr.DiscoveryArn != nil {
				if a, err = adapterhelpers.ParseARN(*cr.DiscoveryArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "servicediscovery-service",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *cr.DiscoveryArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// These are tightly linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	if service.NetworkConfiguration != nil {
		if service.NetworkConfiguration.AwsvpcConfiguration != nil {
			for _, subnet := range service.NetworkConfiguration.AwsvpcConfiguration.Subnets {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  subnet,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the subnet will affect the service
						In: true,
						// The service shouldn't affect the subnet
						Out: false,
					},
				})
			}

			for _, sg := range service.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  sg,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the security group will affect the service
						In: true,
						// The service shouldn't affect the security group
						Out: false,
					},
				})
			}
		}
	}

	for _, id := range taskSetIds {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ecs-task-set",
				Method: sdp.QueryMethod_GET,
				Query:  id,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				In:  true,
				Out: true,
			},
		})
	}

	return &item, nil
}

func serviceListFuncOutputMapper(output *ecs.ListServicesOutput, input *ecs.ListServicesInput) ([]*ecs.DescribeServicesInput, error) {
	inputs := make([]*ecs.DescribeServicesInput, 0)

	var a *adapterhelpers.ARN
	var err error

	for _, arn := range output.ServiceArns {
		a, err = adapterhelpers.ParseARN(arn)

		if err != nil {
			continue
		}

		sections := strings.Split(a.Resource, "/")

		if len(sections) != 3 {
			return nil, fmt.Errorf("could not split into 3 sections on '/': %v", a.Resource)
		}

		inputs = append(inputs, &ecs.DescribeServicesInput{
			Cluster: &sections[1],
			Services: []string{
				sections[2],
			},
			Include: ServiceIncludeFields,
		})
	}

	return inputs, nil
}

func NewECSServiceAdapter(client ECSClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*ecs.ListServicesInput, *ecs.ListServicesOutput, *ecs.DescribeServicesInput, *ecs.DescribeServicesOutput, ECSClient, *ecs.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*ecs.ListServicesInput, *ecs.ListServicesOutput, *ecs.DescribeServicesInput, *ecs.DescribeServicesOutput, ECSClient, *ecs.Options]{
		ItemType:        "ecs-service",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         serviceGetFunc,
		DisableList:     true,
		AdapterMetadata: ecsServiceAdapterMetadata,
		GetInputMapper: func(scope, query string) *ecs.DescribeServicesInput {
			// We are using a custom id of {clusterName}/{id} e.g.
			// ecs-template-ECSCluster-8nS0WOLbs3nZ/ecs-template-service-i0mQKzkhDI2C
			sections := strings.Split(query, "/")

			if len(sections) != 2 {
				return nil
			}

			return &ecs.DescribeServicesInput{
				Services: []string{
					sections[1],
				},
				Cluster: &sections[0],
				Include: ServiceIncludeFields,
			}
		},
		ListInput: &ecs.ListServicesInput{},
		ListFuncPaginatorBuilder: func(client ECSClient, input *ecs.ListServicesInput) adapterhelpers.Paginator[*ecs.ListServicesOutput, *ecs.Options] {
			return ecs.NewListServicesPaginator(client, input)
		},
		SearchInputMapper: func(scope, query string) (*ecs.ListServicesInput, error) {
			// Custom search by cluster
			return &ecs.ListServicesInput{
				Cluster: adapterhelpers.PtrString(query),
			}, nil
		},
		ListFuncOutputMapper: serviceListFuncOutputMapper,
	}
}

var ecsServiceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ecs-service",
	DescriptiveName: "ECS Service",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an ECS service by full name ({clusterName}/{id})",
		SearchDescription: "Search for ECS services by cluster",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "aws_ecs_service.cluster_name",
		},
	},
	PotentialLinks: []string{"ecs-cluster", "elbv2-target-group", "servicediscovery-service", "ecs-task-definition", "ecs-capacity-provider", "ec2-subnet", "ecs-security-group", "dns"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
