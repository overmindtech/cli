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

func (t *ecsTestClient) DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	return &ecs.DescribeTasksOutput{
		Tasks: []types.Task{
			{
				Attachments: []types.Attachment{
					{
						Id:     new("id"), // link?
						Status: new("OK"),
						Type:   new("ElasticNetworkInterface"),
					},
				},
				Attributes: []types.Attribute{
					{
						Name:  new("ecs.cpu-architecture"),
						Value: new("x86_64"),
					},
				},
				AvailabilityZone:     new("eu-west-1c"),
				ClusterArn:           new("arn:aws:ecs:eu-west-1:052392120703:cluster/test-ECSCluster-Bt4SqcM3CURk"), // link
				Connectivity:         types.ConnectivityConnected,
				ConnectivityAt:       new(time.Now()),
				ContainerInstanceArn: new("arn:aws:ecs:eu-west-1:052392120703:container-instance/test-ECSCluster-Bt4SqcM3CURk/4b5c1d7dbb6746b38ada1b97b1866f6a"), // link
				Containers: []types.Container{
					{
						ContainerArn:      new("arn:aws:ecs:eu-west-1:052392120703:container/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2/39a3ede1-1b28-472e-967a-d87d691f65e0"),
						TaskArn:           new("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
						Name:              new("busybox"),
						Image:             new("busybox"),
						RuntimeId:         new("7c158f5c2711416cbb6e653ad90997346489c9722c59d1115ad2121dd040748e"),
						LastStatus:        new("RUNNING"),
						NetworkBindings:   []types.NetworkBinding{},
						NetworkInterfaces: []types.NetworkInterface{},
						HealthStatus:      types.HealthStatusUnknown,
						Cpu:               new("10"),
						Memory:            new("200"),
					},
					{
						ContainerArn: new("arn:aws:ecs:eu-west-1:052392120703:container/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2/8f3db814-6b39-4cc0-9d0a-a7d5702175eb"),
						TaskArn:      new("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
						Name:         new("simple-app"),
						Image:        new("httpd:2.4"),
						RuntimeId:    new("7316b64efb397cececce7cc5f39c6d48ab454f904cc80009aef5ed01ebdb1333"),
						LastStatus:   new("RUNNING"),
						NetworkBindings: []types.NetworkBinding{
							{
								BindIP:        new("0.0.0.0"), // Link? NetworkSocket?
								ContainerPort: new(int32(80)),
								HostPort:      new(int32(32768)),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						NetworkInterfaces: []types.NetworkInterface{
							{
								AttachmentId:       new("attachmentId"),
								Ipv6Address:        new("2001:db8:3333:4444:5555:6666:7777:8888"), // link
								PrivateIpv4Address: new("10.0.0.1"),                               // link
							},
						},
						HealthStatus: types.HealthStatusUnknown,
						Cpu:          new("10"),
						Memory:       new("300"),
					},
				},
				Cpu:                  new("20"),
				CreatedAt:            new(time.Now()),
				DesiredStatus:        new("RUNNING"),
				EnableExecuteCommand: false,
				Group:                new("service:test-service-lszmaXSqRKuF"),
				HealthStatus:         types.HealthStatusUnknown,
				LastStatus:           new("RUNNING"),
				LaunchType:           types.LaunchTypeEc2,
				Memory:               new("500"),
				Overrides: &types.TaskOverride{
					ContainerOverrides: []types.ContainerOverride{
						{
							Name: new("busybox"),
						},
						{
							Name: new("simple-app"),
						},
					},
					InferenceAcceleratorOverrides: []types.InferenceAcceleratorOverride{},
				},
				PullStartedAt:     new(time.Now()),
				PullStoppedAt:     new(time.Now()),
				StartedAt:         new(time.Now()),
				StartedBy:         new("ecs-svc/0710912874193920929"),
				Tags:              []types.Tag{},
				TaskArn:           new("arn:aws:ecs:eu-west-1:052392120703:task/test-ECSCluster-Bt4SqcM3CURk/2ffd7ed376c841bcb0e6795ddb6e72e2"),
				TaskDefinitionArn: new("arn:aws:ecs:eu-west-1:052392120703:task-definition/test-ecs-demo-app:1"), // link
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
			return
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

	adapter := NewECSTaskAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
