package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *ecsTestClient) DescribeContainerInstances(ctx context.Context, params *ecs.DescribeContainerInstancesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error) {
	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []types.ContainerInstance{
			{
				ContainerInstanceArn: adapterhelpers.PtrString("arn:aws:ecs:eu-west-1:052392120703:container-instance/ecs-template-ECSCluster-8nS0WOLbs3nZ/50e9bf71ed57450ca56293cc5a042886"),
				Ec2InstanceId:        adapterhelpers.PtrString("i-0e778f25705bc0c84"), // link
				Version:              4,
				VersionInfo: &types.VersionInfo{
					AgentVersion:  adapterhelpers.PtrString("1.47.0"),
					AgentHash:     adapterhelpers.PtrString("1489adfa"),
					DockerVersion: adapterhelpers.PtrString("DockerVersion: 19.03.6-ce"),
				},
				RemainingResources: []types.Resource{
					{
						Name:         adapterhelpers.PtrString("CPU"),
						Type:         adapterhelpers.PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2028,
					},
					{
						Name:         adapterhelpers.PtrString("MEMORY"),
						Type:         adapterhelpers.PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7474,
					},
					{
						Name:         adapterhelpers.PtrString("PORTS"),
						Type:         adapterhelpers.PtrString("STRINGSET"),
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
						Name:           adapterhelpers.PtrString("PORTS_UDP"),
						Type:           adapterhelpers.PtrString("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				RegisteredResources: []types.Resource{
					{
						Name:         adapterhelpers.PtrString("CPU"),
						Type:         adapterhelpers.PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 2048,
					},
					{
						Name:         adapterhelpers.PtrString("MEMORY"),
						Type:         adapterhelpers.PtrString("INTEGER"),
						DoubleValue:  0.0,
						LongValue:    0,
						IntegerValue: 7974,
					},
					{
						Name:         adapterhelpers.PtrString("PORTS"),
						Type:         adapterhelpers.PtrString("STRINGSET"),
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
						Name:           adapterhelpers.PtrString("PORTS_UDP"),
						Type:           adapterhelpers.PtrString("STRINGSET"),
						DoubleValue:    0.0,
						LongValue:      0,
						IntegerValue:   0,
						StringSetValue: []string{},
					},
				},
				Status:            adapterhelpers.PtrString("ACTIVE"),
				AgentConnected:    true,
				RunningTasksCount: 1,
				PendingTasksCount: 0,
				Attributes: []types.Attribute{
					{
						Name: adapterhelpers.PtrString("ecs.capability.secrets.asm.environment-variables"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.capability.branch-cni-plugin-version"),
						Value: adapterhelpers.PtrString("a21d3a41-"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.ami-id"),
						Value: adapterhelpers.PtrString("ami-0c9ef930279337028"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.secrets.asm.bootstrap.log-driver"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-eia.optimized-cpu"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.none"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.ecr-endpoint"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.docker-plugin.local"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-cpu-mem-limit"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.secrets.ssm.bootstrap.log-driver"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.efsAuth"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.full-sync"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.30"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.31"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.32"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.fluentd"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.firelens.options.config.file"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.availability-zone"),
						Value: adapterhelpers.PtrString("eu-west-1a"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.aws-appmesh"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.awslogs"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.24"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-eni-trunking"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.25"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.26"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.27"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.privileged-container"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.28"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.29"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.cpu-architecture"),
						Value: adapterhelpers.PtrString("x86_64"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.ecr-auth"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.firelens.fluentbit"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.20"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.os-type"),
						Value: adapterhelpers.PtrString("linux"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.21"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.22"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.23"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-eia"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.private-registry-authentication.secretsmanager"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.syslog"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.awsfirelens"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.firelens.options.config.s3"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.logging-driver.json-file"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.execution-role-awslogs"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.vpc-id"),
						Value: adapterhelpers.PtrString("vpc-0e120717a7263de70"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.17"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.18"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.docker-remote-api.1.19"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.docker-plugin.amazon-ecs-volume-plugin"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-eni"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.firelens.fluentd"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.efs"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.execution-role-ecr-pull"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.task-eni.ipv6"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.container-health-check"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.subnet-id"),
						Value: adapterhelpers.PtrString("subnet-0bfdb717a234c01b3"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.instance-type"),
						Value: adapterhelpers.PtrString("t2.large"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.task-iam-role-network-host"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.container-ordering"),
					},
					{
						Name:  adapterhelpers.PtrString("ecs.capability.cni-plugin-version"),
						Value: adapterhelpers.PtrString("55b2ae77-2020.09.0"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.env-files.s3"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.pid-ipc-namespace-sharing"),
					},
					{
						Name: adapterhelpers.PtrString("ecs.capability.secrets.ssm.environment-variables"),
					},
					{
						Name: adapterhelpers.PtrString("com.amazonaws.ecs.capability.task-iam-role"),
					},
				},
				RegisteredAt:         adapterhelpers.PtrTime(time.Now()),
				Attachments:          []types.Attachment{}, // There is probably an opportunity for some links here but I don't have example data
				Tags:                 []types.Tag{},
				AgentUpdateStatus:    types.AgentUpdateStatusFailed,
				CapacityProviderName: adapterhelpers.PtrString("name"),
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

	tests := adapterhelpers.QueryTests{
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

	adapter := NewECSContainerInstanceAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
