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

func (t *ecsTestClient) DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	return &ecs.DescribeClustersOutput{
		Clusters: []types.Cluster{
			{
				ClusterArn:                        adapterhelpers.PtrString("arn:aws:ecs:eu-west-2:052392120703:cluster/default"),
				ClusterName:                       adapterhelpers.PtrString("default"),
				Status:                            adapterhelpers.PtrString("ACTIVE"),
				RegisteredContainerInstancesCount: 0,
				RunningTasksCount:                 1,
				PendingTasksCount:                 0,
				ActiveServicesCount:               1,
				Statistics: []types.KeyValuePair{
					{
						Name:  adapterhelpers.PtrString("key"),
						Value: adapterhelpers.PtrString("value"),
					},
				},
				Tags: []types.Tag{},
				Settings: []types.ClusterSetting{
					{
						Name:  types.ClusterSettingNameContainerInsights,
						Value: adapterhelpers.PtrString("ENABLED"),
					},
				},
				CapacityProviders: []string{
					"test",
				},
				DefaultCapacityProviderStrategy: []types.CapacityProviderStrategyItem{
					{
						CapacityProvider: adapterhelpers.PtrString("provider"),
						Base:             10,
						Weight:           100,
					},
				},
				Attachments: []types.Attachment{
					{
						Id:     adapterhelpers.PtrString("1c1f9cf4-461c-4072-aab2-e2dd346c53e1"),
						Type:   adapterhelpers.PtrString("as_policy"),
						Status: adapterhelpers.PtrString("CREATED"),
						Details: []types.KeyValuePair{
							{
								Name:  adapterhelpers.PtrString("capacityProviderName"),
								Value: adapterhelpers.PtrString("test"),
							},
							{
								Name:  adapterhelpers.PtrString("scalingPolicyName"),
								Value: adapterhelpers.PtrString("ECSManagedAutoScalingPolicy-d2f110eb-20a6-4278-9c1c-47d98e21b1ed"),
							},
						},
					},
				},
				AttachmentsStatus: adapterhelpers.PtrString("UPDATE_COMPLETE"),
				Configuration: &types.ClusterConfiguration{
					ExecuteCommandConfiguration: &types.ExecuteCommandConfiguration{
						KmsKeyId: adapterhelpers.PtrString("id"),
						LogConfiguration: &types.ExecuteCommandLogConfiguration{
							CloudWatchEncryptionEnabled: true,
							CloudWatchLogGroupName:      adapterhelpers.PtrString("cloud-watch-name"),
							S3BucketName:                adapterhelpers.PtrString("s3-name"),
							S3EncryptionEnabled:         true,
							S3KeyPrefix:                 adapterhelpers.PtrString("prod"),
						},
					},
				},
				ServiceConnectDefaults: &types.ClusterServiceConnectDefaults{
					Namespace: adapterhelpers.PtrString("prod"),
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

	tests := adapterhelpers.QueryTests{
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

	adapter := NewECSClusterAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
