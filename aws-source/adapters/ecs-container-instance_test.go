package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func (t *ecsTestClient) DescribeContainerInstances(ctx context.Context, params *ecs.DescribeContainerInstancesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error) {
	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []types.ContainerInstance{
			{
				ContainerInstanceArn: PtrString("arn:aws:ecs:eu-west-1:052392120703:container-instance/ecs-template-ECSCluster-8nS0WOLbs3nZ/50e9bf71ed57450ca56293cc5a042886"),
				Ec2InstanceId:        PtrString("i-0e778f25705bc0c84"), // link
				Version:              4,
				VersionInfo: &types.VersionInfo{
					AgentVersion:  PtrString("1.47.0"),
					AgentHash:     PtrString("1489adfa"),
					DockerVersion: PtrString("DockerVersion: 19.03.6-ce"),
				},
				RemainingResources: []types.Resource{
					{
						Name:         PtrString("CPU"),
						Type:         PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2028,
					},
					{
						Name:         PtrString("MEMORY"),
						Type:         PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7474,
					},
					{
						Name:         PtrString("PORTS"),
						Type:         PtrString("STRINGSET"),
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
						Name:           PtrString("PORTS_UDP"),
						Type:           PtrString("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				RegisteredResources: []types.Resource{
					{
						Name:         PtrString("CPU"),
						Type:         PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2048,
					},
					{
						Name:         PtrString("MEMORY"),
						Type:         PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7974,
					},
					{
						Name:         PtrString("PORTS"),
						Type:         PtrString("STRINGSET"),
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
						Name:           PtrString("PORTS_UDP"),
						Type:           PtrString("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				Status:            PtrString("ACTIVE"),
				AgentConnected:    true,
				RunningTasksCount: 1,
				PendingTasksCount: 0,
				Attributes: []types.Attribute{
					{
						Name: PtrString("ecs.capability.secrets.asm.environment-variables"),
					},
					{
						Name:  PtrString("ecs.capability.branch-cni-plugin-version"),
						Value: PtrString("a21d3a41-"),
					},
					{
						Name:  PtrString("ecs.ami-id"),
						Value: PtrString("ami-0c9ef930279337028"),
					},
					{
						Name: PtrString("ecs.capability.secrets.asm.bootstrap.log-driver"),
					},
					{
						Name: PtrString("ecs.capability.task-eia.optimized-cpu"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.none"),
					},
					{
						Name: PtrString("ecs.capability.ecr-endpoint"),
					},
					{
						Name: PtrString("ecs.capability.docker-plugin.local"),
					},
					{
						Name: PtrString("ecs.capability.task-cpu-mem-limit"),
					},
					{
						Name: PtrString("ecs.capability.secrets.ssm.bootstrap.log-driver"),
					},
					{
						Name: PtrString("ecs.capability.efsAuth"),
					},
					{
						Name: PtrString("ecs.capability.full-sync"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.30"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.31"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.32"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.fluentd"),
					},
					{
						Name: PtrString("ecs.capability.firelens.options.config.file"),
					},
					{
						Name:  PtrString("ecs.availability-zone"),
						Value: PtrString("eu-west-1a"),
					},
					{
						Name: PtrString("ecs.capability.aws-appmesh"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.awslogs"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.24"),
					},
					{
						Name: PtrString("ecs.capability.task-eni-trunking"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.25"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.26"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.27"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.privileged-container"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.28"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.29"),
					},
					{
						Name:  PtrString("ecs.cpu-architecture"),
						Value: PtrString("x86_64"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.ecr-auth"),
					},
					{
						Name: PtrString("ecs.capability.firelens.fluentbit"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.20"),
					},
					{
						Name:  PtrString("ecs.os-type"),
						Value: PtrString("linux"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.21"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.22"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.23"),
					},
					{
						Name: PtrString("ecs.capability.task-eia"),
					},
					{
						Name: PtrString("ecs.capability.private-registry-authentication.secretsmanager"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.syslog"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.awsfirelens"),
					},
					{
						Name: PtrString("ecs.capability.firelens.options.config.s3"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.logging-driver.json-file"),
					},
					{
						Name: PtrString("ecs.capability.execution-role-awslogs"),
					},
					{
						Name:  PtrString("ecs.vpc-id"),
						Value: PtrString("vpc-0e120717a7263de70"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.17"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.18"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.19"),
					},
					{
						Name: PtrString("ecs.capability.docker-plugin.amazon-ecs-volume-plugin"),
					},
					{
						Name: PtrString("ecs.capability.task-eni"),
					},
					{
						Name: PtrString("ecs.capability.firelens.fluentd"),
					},
					{
						Name: PtrString("ecs.capability.efs"),
					},
					{
						Name: PtrString("ecs.capability.execution-role-ecr-pull"),
					},
					{
						Name: PtrString("ecs.capability.task-eni.ipv6"),
					},
					{
						Name: PtrString("ecs.capability.container-health-check"),
					},
					{
						Name:  PtrString("ecs.subnet-id"),
						Value: PtrString("subnet-0bfdb717a234c01b3"),
					},
					{
						Name:  PtrString("ecs.instance-type"),
						Value: PtrString("t2.large"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.task-iam-role-network-host"),
					},
					{
						Name: PtrString("ecs.capability.container-ordering"),
					},
					{
						Name:  PtrString("ecs.capability.cni-plugin-version"),
						Value: PtrString("55b2ae77-2020.09.0"),
					},
					{
						Name: PtrString("ecs.capability.env-files.s3"),
					},
					{
						Name: PtrString("ecs.capability.pid-ipc-namespace-sharing"),
					},
					{
						Name: PtrString("ecs.capability.secrets.ssm.environment-variables"),
					},
					{
						Name: PtrString("com.amazonaws.ecs.capability.task-iam-role"),
					},
				},
				RegisteredAt:         PtrTime(time.Now()),
				Attachments:          []types.Attachment{}, // There is probably an opportunity for some links here but I don't have example data
				Tags:                 []types.Tag{},
				AgentUpdateStatus:    types.AgentUpdateStatusFailed,
				CapacityProviderName: PtrString("name"),
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
