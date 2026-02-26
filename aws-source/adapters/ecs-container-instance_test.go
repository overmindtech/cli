package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func (t *ecsTestClient) DescribeContainerInstances(ctx context.Context, params *ecs.DescribeContainerInstancesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error) {
	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []types.ContainerInstance{
			{
				ContainerInstanceArn: new("arn:aws:ecs:eu-west-1:052392120703:container-instance/ecs-template-ECSCluster-8nS0WOLbs3nZ/50e9bf71ed57450ca56293cc5a042886"),
				Ec2InstanceId:        new("i-0e778f25705bc0c84"), // link
				Version:              4,
				VersionInfo: &types.VersionInfo{
					AgentVersion:  new("1.47.0"),
					AgentHash:     new("1489adfa"),
					DockerVersion: new("DockerVersion: 19.03.6-ce"),
				},
				RemainingResources: []types.Resource{
					{
						Name:         new("CPU"),
						Type:         new("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2028,
					},
					{
						Name:         new("MEMORY"),
						Type:         new("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7474,
					},
					{
						Name:         new("PORTS"),
						Type:         new("STRINGSET"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 0,
						StringSetValue: []string{
							"22",
							"2376",
							"2375",
							"51678",
							"51679",
						},
					},
					{
						Name:           new("PORTS_UDP"),
						Type:           new("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				RegisteredResources: []types.Resource{
					{
						Name:         new("CPU"),
						Type:         new("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2048,
					},
					{
						Name:         new("MEMORY"),
						Type:         new("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7974,
					},
					{
						Name:         new("PORTS"),
						Type:         new("STRINGSET"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 0,
						StringSetValue: []string{
							"22",
							"2376",
							"2375",
							"51678",
							"51679",
						},
					},
					{
						Name:           new("PORTS_UDP"),
						Type:           new("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				Status:            new("ACTIVE"),
				AgentConnected:    true,
				RunningTasksCount: 1,
				PendingTasksCount: 0,
				Attributes: []types.Attribute{
					{
						Name: new("ecs.capability.secrets.asm.environment-variables"),
					},
					{
						Name:  new("ecs.capability.branch-cni-plugin-version"),
						Value: new("a21d3a41-"),
					},
					{
						Name:  new("ecs.ami-id"),
						Value: new("ami-0c9ef930279337028"),
					},
					{
						Name: new("ecs.capability.secrets.asm.bootstrap.log-driver"),
					},
					{
						Name: new("ecs.capability.task-eia.optimized-cpu"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.none"),
					},
					{
						Name: new("ecs.capability.ecr-endpoint"),
					},
					{
						Name: new("ecs.capability.docker-plugin.local"),
					},
					{
						Name: new("ecs.capability.task-cpu-mem-limit"),
					},
					{
						Name: new("ecs.capability.secrets.ssm.bootstrap.log-driver"),
					},
					{
						Name: new("ecs.capability.efsAuth"),
					},
					{
						Name: new("ecs.capability.full-sync"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.30"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.31"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.32"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.fluentd"),
					},
					{
						Name: new("ecs.capability.firelens.options.config.file"),
					},
					{
						Name:  new("ecs.availability-zone"),
						Value: new("eu-west-1a"),
					},
					{
						Name: new("ecs.capability.aws-appmesh"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.awslogs"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.24"),
					},
					{
						Name: new("ecs.capability.task-eni-trunking"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.25"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.26"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.27"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.privileged-container"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.28"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.29"),
					},
					{
						Name:  new("ecs.cpu-architecture"),
						Value: new("x86_64"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.ecr-auth"),
					},
					{
						Name: new("ecs.capability.firelens.fluentbit"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.20"),
					},
					{
						Name:  new("ecs.os-type"),
						Value: new("linux"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.21"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.22"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.23"),
					},
					{
						Name: new("ecs.capability.task-eia"),
					},
					{
						Name: new("ecs.capability.private-registry-authentication.secretsmanager"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.syslog"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.awsfirelens"),
					},
					{
						Name: new("ecs.capability.firelens.options.config.s3"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.logging-driver.json-file"),
					},
					{
						Name: new("ecs.capability.execution-role-awslogs"),
					},
					{
						Name:  new("ecs.vpc-id"),
						Value: new("vpc-0e120717a7263de70"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.17"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.18"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.19"),
					},
					{
						Name: new("ecs.capability.docker-plugin.amazon-ecs-volume-plugin"),
					},
					{
						Name: new("ecs.capability.task-eni"),
					},
					{
						Name: new("ecs.capability.firelens.fluentd"),
					},
					{
						Name: new("ecs.capability.efs"),
					},
					{
						Name: new("ecs.capability.execution-role-ecr-pull"),
					},
					{
						Name: new("ecs.capability.task-eni.ipv6"),
					},
					{
						Name: new("ecs.capability.container-health-check"),
					},
					{
						Name:  new("ecs.subnet-id"),
						Value: new("subnet-0bfdb717a234c01b3"),
					},
					{
						Name:  new("ecs.instance-type"),
						Value: new("t2.large"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.task-iam-role-network-host"),
					},
					{
						Name: new("ecs.capability.container-ordering"),
					},
					{
						Name:  new("ecs.capability.cni-plugin-version"),
						Value: new("55b2ae77-2020.09.0"),
					},
					{
						Name: new("ecs.capability.env-files.s3"),
					},
					{
						Name: new("ecs.capability.pid-ipc-namespace-sharing"),
					},
					{
						Name: new("ecs.capability.secrets.ssm.environment-variables"),
					},
					{
						Name: new("com.amazonaws.ecs.capability.task-iam-role"),
					},
				},
				RegisteredAt:         new(time.Now()),
				Attachments:          []types.Attachment{}, // There is probably an opportunity for some links here but I don't have example data
				Tags:                 []types.Tag{},
				AgentUpdateStatus:    types.AgentUpdateStatusFailed,
				CapacityProviderName: new("name"),
				HealthStatus: &types.ContainerInstanceHealthStatus{
					OverallStatus: types.InstanceHealthCheckStateImpaired,
				},
			},
		},
	}, nil
}

func (t *ecsTestClient) ListContainerInstances(context.Context, *ecs.ListContainerInstancesInput, ...func(*ecs.Options)) (*ecs.ListContainerInstancesOutput, error) {
	return &ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: []string{
			"arn:aws:ecs:eu-west-1:052392120703:container-instance/ecs-template-ECSCluster-8nS0WOLbs3nZ/50e9bf71ed57450ca56293cc5a042886",
		},
	}, nil
}

func TestContainerInstanceGetFunc(t *testing.T) {
	item, err := containerInstanceGetFunc(context.Background(), &ecsTestClient{}, "foo", &ecs.DescribeContainerInstancesInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-0e778f25705bc0c84",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewECSContainerInstanceAdapter(t *testing.T) {
	client, account, region := ecsGetAutoConfig(t)

	adapter := NewECSContainerInstanceAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
