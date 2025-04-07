package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// TaskIncludeFields Fields that we want included by default
var TaskIncludeFields = []types.TaskField{
	types.TaskFieldTags,
}

func taskGetFunc(ctx context.Context, client ECSClient, scope string, input *ecs.DescribeTasksInput) (*sdp.Item, error) {
	out, err := client.DescribeTasks(ctx, input)

	if err != nil {
		return nil, err
	}

	if len(out.Tasks) != 1 {
		return nil, fmt.Errorf("expected 1 task, got %v", len(out.Tasks))
	}

	task := out.Tasks[0]

	attributes, err := adapterhelpers.ToAttributesWithExclude(task, "tags")

	if err != nil {
		return nil, err
	}

	if task.TaskArn == nil {
		return nil, errors.New("task has nil ARN")
	}

	a, err := adapterhelpers.ParseARN(*task.TaskArn)

	if err != nil {
		return nil, err
	}

	// Create unique attribute in the format {clusterName}/{id}
	// test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2
	attributes.Set("Id", a.ResourceID())

	item := sdp.Item{
		Type:            "ecs-task",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            ecsTagsToMap(task.Tags),
	}

	switch task.HealthStatus {
	case types.HealthStatusHealthy:
		item.Health = sdp.Health_HEALTH_OK.Enum()
	case types.HealthStatusUnhealthy:
		item.Health = sdp.Health_HEALTH_ERROR.Enum()
	case types.HealthStatusUnknown:
		item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	for _, attachment := range task.Attachments {
		if attachment.Type != nil {
			if *attachment.Type == "ElasticNetworkInterface" {
				if attachment.Id != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-network-interface",
							Method: sdp.QueryMethod_GET,
							Query:  *attachment.Id,
							Scope:  scope,
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

	if task.ClusterArn != nil {
		if a, err = adapterhelpers.ParseARN(*task.ClusterArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ecs-cluster",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *task.ClusterArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The cluster can affect the task
					In: true,
					// The task can't affect the cluster
					Out: false,
				},
			})
		}
	}

	if task.ContainerInstanceArn != nil {
		if a, err = adapterhelpers.ParseARN(*task.ContainerInstanceArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ecs-container-instance",
					Method: sdp.QueryMethod_GET,
					Query:  a.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The container instance can affect the task
					In: true,
					// The task can't affect the container instance
					Out: false,
				},
			})
		}
	}

	for _, container := range task.Containers {
		for _, ni := range container.NetworkInterfaces {
			if ni.Ipv6Address != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ni.Ipv6Address,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked
						In:  true,
						Out: true,
					},
				})
			}

			if ni.PrivateIpv4Address != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ni.PrivateIpv4Address,
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
	}

	if task.TaskDefinitionArn != nil {
		if a, err = adapterhelpers.ParseARN(*task.TaskDefinitionArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ecs-task-definition",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *task.TaskDefinitionArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The task definition can affect the task
					In: true,
					// The task can't affect the task definition
					Out: false,
				},
			})
		}
	}

	return &item, nil
}

func taskGetInputMapper(scope, query string) *ecs.DescribeTasksInput {
	// `id` is {clusterName}/{id} so split on '/'
	sections := strings.Split(query, "/")

	if len(sections) != 2 {
		return nil
	}

	return &ecs.DescribeTasksInput{
		Tasks: []string{
			sections[1],
		},
		Cluster: adapterhelpers.PtrString(sections[0]),
		Include: TaskIncludeFields,
	}
}

func tasksListFuncOutputMapper(output *ecs.ListTasksOutput, input *ecs.ListTasksInput) ([]*ecs.DescribeTasksInput, error) {
	inputs := make([]*ecs.DescribeTasksInput, 0)

	for _, taskArn := range output.TaskArns {
		if a, err := adapterhelpers.ParseARN(taskArn); err == nil {
			// split the cluster name out
			sections := strings.Split(a.ResourceID(), "/")

			if len(sections) != 2 {
				continue
			}

			inputs = append(inputs, &ecs.DescribeTasksInput{
				Tasks: []string{
					sections[1],
				},
				Cluster: &sections[0],
				Include: TaskIncludeFields,
			})
		}
	}

	return inputs, nil
}

func NewECSTaskAdapter(client ECSClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*ecs.ListTasksInput, *ecs.ListTasksOutput, *ecs.DescribeTasksInput, *ecs.DescribeTasksOutput, ECSClient, *ecs.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*ecs.ListTasksInput, *ecs.ListTasksOutput, *ecs.DescribeTasksInput, *ecs.DescribeTasksOutput, ECSClient, *ecs.Options]{
		ItemType:        "ecs-task",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         taskGetFunc,
		AdapterMetadata: ecsTaskAdapterMetadata,
		ListInput:       &ecs.ListTasksInput{},
		GetInputMapper:  taskGetInputMapper,
		DisableList:     true,
		SearchInputMapper: func(scope, query string) (*ecs.ListTasksInput, error) {
			// Search by cluster
			return &ecs.ListTasksInput{
				Cluster: adapterhelpers.PtrString(query),
			}, nil
		},
		ListFuncPaginatorBuilder: func(client ECSClient, input *ecs.ListTasksInput) adapterhelpers.Paginator[*ecs.ListTasksOutput, *ecs.Options] {
			return ecs.NewListTasksPaginator(client, input)
		},
		ListFuncOutputMapper: tasksListFuncOutputMapper,
	}
}

var ecsTaskAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ecs-task",
	DescriptiveName: "ECS Task",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an ECS task by ID",
		SearchDescription: "Search for ECS tasks by cluster",
	},
	PotentialLinks: []string{"ecs-cluster", "ecs-container-instance", "ecs-task-definition", "ec2-network-interface", "ip"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
