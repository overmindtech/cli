package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *ecsTestClient) DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	return &ecs.DescribeTasksOutput{
		Tasks: []types.Task{
			{
				Attachments: []types.Attachment{
					{
						Id:     PtrString("id"), // link?
						Status: PtrString("OK"),
						Type:   PtrString("ElasticNetworkInterface"),
					},
				},
				Attributes: []types.Attribute{
					{
						Name:  PtrString("ecs.cpu-architecture"),
						Value: PtrString("x86_64"),
					},
				},
				AvailabilityZone:     PtrString("eu-west-1c"),
				ClusterArn:           PtrString("arn:aws:ecs:eu-west-1:052392120703:cluster/test-ECSCluster-Bt4SqcM3CURk"), // link
				Connectivity:         types.ConnectivityConnected,
				ConnectivityAt:       PtrTime(time.Now()),
				ContainerInstanceArn: PtrString("arn:aws:ecs:eu-west-1:052392120703:container-instance/test-ECSCluster-Bt4SqcM3CURk/4b5c1d7dbb6746b38ada1b97b1866f6a"), // link
				Containers: []types.Container{
					{
						ContainerArn:      PtrString("arn:aws:ecs:eu-west-1:052392120703:container/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2/39a3ede1-1b28-472e-967a-d87d691f65e0"),
						TaskArn:           PtrString("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
						Name:              PtrString("busybox"),
						Image:             PtrString("busybox"),
						RuntimeId:         PtrString("7c158f5c2711416cbb6e653ad90997346489c9722c59d1115ad2121dd040748e"),
						LastStatus:        PtrString("RUNNING"),
						NetworkBindings:   []types.NetworkBinding{},
						NetworkInterfaces: []types.NetworkInterface{},
						HealthStatus:      types.HealthStatusUnknown,
						Cpu:               PtrString("10"),
						Memory:            PtrString("200"),
					},
					{
						ContainerArn: PtrString("arn:aws:ecs:eu-west-1:052392120703:container/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2/8f3db814-6b39-4cc0-9d0a-a7d5702175eb"),
						TaskArn:      PtrString("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
						Name:         PtrString("simple-app"),
						Image:        PtrString("httpd:2.4"),
						RuntimeId:    PtrString("7316b64efb397cececce7cc5f39c6d48ab454f904cc80009aef5ed01ebdb1333"),
						LastStatus:   PtrString("RUNNING"),
						NetworkBindings: []types.NetworkBinding{
							{
								BindIP:        PtrString("0.0.0.0"), // Link? NetworkSocket?
								ContainerPort: PtrInt32(80),
								HostPort:      PtrInt32(32768),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						NetworkInterfaces: []types.NetworkInterface{
							{
								AttachmentId:       PtrString("attachmentId"),
								Ipv6Address:        PtrString("2001:db8:3333:4444:5555:6666:7777:8888"), // link
								PrivateIpv4Address: PtrString("10.0.0.1"),                               // link
							},
						},
						HealthStatus: types.HealthStatusUnknown,
						Cpu:          PtrString("10"),
						Memory:       PtrString("300"),
					},
				},
				Cpu:                  PtrString("20"),
				CreatedAt:            PtrTime(time.Now()),
				DesiredStatus:        PtrString("RUNNING"),
				EnableExecuteCommand: false,
				Group:                PtrString("service:test-service-lszmaXSqRKuF"),
				HealthStatus:         types.HealthStatusUnknown,
				LastStatus:           PtrString("RUNNING"),
				LaunchType:           types.LaunchTypeEc2,
				Memory:               PtrString("500"),
				Overrides: &types.TaskOverride{
					ContainerOverrides: []types.ContainerOverride{
						{
							Name: PtrString("busybox"),
						},
						{
							Name: PtrString("simple-app"),
						},
					},
					InferenceAcceleratorOverrides: []types.InferenceAcceleratorOverride{},
				},
				PullStartedAt:     PtrTime(time.Now()),
				PullStoppedAt:     PtrTime(time.Now()),
				StartedAt:         PtrTime(time.Now()),
				StartedBy:         PtrString("ecs-svc/0710912874193920929"),
				Tags:              []types.Tag{},
				TaskArn:           PtrString("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
				TaskDefinitionArn: PtrString("arn:aws:ecs:eu-west-1:052392120703:task-definition/test-ecs-demo-app:1"), // link
				Version:           3,
				EphemeralStorage: &types.EphemeralStorage{
					SizeInGiB: 1,
				},
			},
		},
	}, nil
}

func (t *ecsTestClient) ListTasks(context.Context, *ecs.ListTasksInput, ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	return &ecs.ListTasksOutput{
		TaskArns: []string{
			"arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2",
		},
	}, nil
}

func TestTaskGetInputMapper(t *testing.T) {
	t.Run("test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2", func(t *testing.T) {
		input := taskGetInputMapper("foo", "test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2")

		if input == nil {
			t.Fatal("input is nil")
		}

		if *input.Cluster != "test-ECSCluster-Bt4SqcM3CURk" {
			t.Errorf("expected cluster to be test-ECSCluster-Bt4SqcM3CURk, got %v", *input.Cluster)
		}

		if input.Tasks[0] != "2ffd7ed376c841bcb0e6795ddb6e72e2" {
			t.Errorf("expected task to be 2ffd7ed376c841bcb0e6795ddb6e72e2, got %v", input.Tasks[0])
		}
	})

	t.Run("2ffd7ed376c841bcb0e6795ddb6e72e2", func(t *testing.T) {
		input := taskGetInputMapper("foo", "2ffd7ed376c841bcb0e6795ddb6e72e2")

		if input != nil {
			t.Error("expected input to be nil")
		}
	})

	t.Run("blah", func(t *testing.T) {
		input := taskGetInputMapper("foo", "blah")

		if input != nil {
			t.Error("expected input to be nil")
		}
	})
}

func TestTasksListFuncOutputMapper(t *testing.T) {
	inputs, err := tasksListFuncOutputMapper(&ecs.ListTasksOutput{
		TaskArns: []string{
			"arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2",
			"bad",
		},
	}, &ecs.ListTasksInput{})

	if err != nil {
		t.Error(err)
	}

	if len(inputs) != 1 {
		t.Fatalf("expected 1 input, got %v", len(inputs))
	}

	if *inputs[0].Cluster != "test-ECSCluster-Bt4SqcM3CURk" {
		t.Errorf("expected cluster to be test-ECSCluster-Bt4SqcM3CURk, got %v", *inputs[0].Cluster)
	}

	if inputs[0].Tasks[0] != "2ffd7ed376c841bcb0e6795ddb6e72e2" {
		t.Errorf("expected task to be 2ffd7ed376c841bcb0e6795ddb6e72e2, got %v", inputs[0].Tasks[0])
	}
}

func TestTaskGetFunc(t *testing.T) {
	item, err := taskGetFunc(context.Background(), &ecsTestClient{}, "foo", &ecs.DescribeTasksInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "ec2-network-interface",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ecs-cluster",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ecs:eu-west-1:052392120703:cluster/test-ECSCluster-Bt4SqcM3CURk",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "ecs-container-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test-ECSCluster-Bt4SqcM3CURk/4b5c1d7dbb6746b38ada1b97b1866f6a",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "2001:db8:3333:4444:5555:6666:7777:8888",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "10.0.0.1",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ecs-task-definition",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ecs:eu-west-1:052392120703:task-definition/test-ecs-demo-app:1",
			ExpectedScope:  "052392120703.eu-west-1",
		},
	}

	tests.Execute(t, item)
}

func TestNewECSTaskAdapter(t *testing.T) {
	client, account, region := ecsGetAutoConfig(t)

	adapter := NewECSTaskAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
