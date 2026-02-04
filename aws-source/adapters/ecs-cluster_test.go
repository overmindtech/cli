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

func (t *ecsTestClient) DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	return &ecs.DescribeClustersOutput{
		Clusters: []types.Cluster{
			{
				ClusterArn:                        PtrString("arn:aws:ecs:eu-west-2:052392120703:cluster/default"),
				ClusterName:                       PtrString("default"),
				Status:                            PtrString("ACTIVE"),
				RegisteredContainerInstancesCount: 0,
				RunningTasksCount:                 1,
				PendingTasksCount:                 0,
				ActiveServicesCount:               1,
				Statistics: []types.KeyValuePair{
					{
						Name:  PtrString("key"),
						Value: PtrString("value"),
					},
				},
				Tags: []types.Tag{},
				Settings: []types.ClusterSetting{
					{
						Name:  types.ClusterSettingNameContainerInsights,
						Value: PtrString("ENABLED"),
					},
				},
				CapacityProviders: []string{
					"test",
				},
				DefaultCapacityProviderStrategy: []types.CapacityProviderStrategyItem{
					{
						CapacityProvider: PtrString("provider"),
						Base:             10,
						Weight:           100,
					},
				},
				Attachments: []types.Attachment{
					{
						Id:     PtrString("1c1f9cf4-461c-4072-aab2-e2dd346c53e1"),
						Type:   PtrString("as_policy"),
						Status: PtrString("CREATED"),
						Details: []types.KeyValuePair{
							{
								Name:  PtrString("capacityProviderName"),
								Value: PtrString("test"),
							},
							{
								Name:  PtrString("scalingPolicyName"),
								Value: PtrString("ECSManagedAutoScalingPolicy-d2f110eb-20a6-4278-9c1c-47d98e21b1ed"),
							},
						},
					},
				},
				AttachmentsStatus: PtrString("UPDATE_COMPLETE"),
				Configuration: &types.ClusterConfiguration{
					ExecuteCommandConfiguration: &types.ExecuteCommandConfiguration{
						KmsKeyId: PtrString("id"),
						LogConfiguration: &types.ExecuteCommandLogConfiguration{
							CloudWatchEncryptionEnabled: true,
							CloudWatchLogGroupName:      PtrString("cloud-watch-name"),
							S3BucketName:                PtrString("s3-name"),
							S3EncryptionEnabled:         true,
							S3KeyPrefix:                 PtrString("prod"),
						},
					},
				},
				ServiceConnectDefaults: &types.ClusterServiceConnectDefaults{
					Namespace: PtrString("prod"),
				},
			},
		},
	}, nil
}

func (t *ecsTestClient) ListClusters(context.Context, *ecs.ListClustersInput, ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &ecs.ListClustersOutput{
		ClusterArns: []string{
			"arn:aws:service:region:account:cluster/name",
		},
	}, nil
}

func TestECSClusterGetFunc(t *testing.T) {
	scope := "123456789012.eu-west-2"
	item, err := ecsClusterGetFunc(context.Background(), &ecsTestClient{}, scope, &ecs.DescribeClustersInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "logs-log-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cloud-watch-name",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "s3-name",
			ExpectedScope:  "123456789012",
		},
		{
			ExpectedType:   "ecs-capacity-provider",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "ecs-container-instance",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "ecs-service",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestECSNewECSClusterAdapter(t *testing.T) {
	client, account, region := ecsGetAutoConfig(t)

	adapter := NewECSClusterAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
