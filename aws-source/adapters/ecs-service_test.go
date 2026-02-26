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

func (t *ecsTestClient) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return &ecs.DescribeServicesOutput{
		Failures: []types.Failure{},
		Services: []types.Service{
			{
				ServiceArn:  new("arn:aws:ecs:eu-west-1:052392120703:service/ecs-template-ECSCluster-8nS0WOLbs3nZ/ecs-template-service-i0mQKzkhDI2C"),
				ServiceName: new("ecs-template-service-i0mQKzkhDI2C"),
				ClusterArn:  new("arn:aws:ecs:eu-west-1:052392120703:cluster/ecs-template-ECSCluster-8nS0WOLbs3nZ"), // link
				LoadBalancers: []types.LoadBalancer{
					{
						TargetGroupArn: new("arn:aws:elasticloadbalancing:eu-west-1:052392120703:targetgroup/ECSTG/0c44b1cdb3437902"), // link
						ContainerName:  new("simple-app"),
						ContainerPort:  new(int32(80)),
					},
				},
				ServiceRegistries: []types.ServiceRegistry{
					{
						ContainerName: new("name"),
						ContainerPort: new(int32(80)),
						Port:          new(int32(80)),
						RegistryArn:   new("arn:aws:service:region:account:type:name"), // link
					},
				},
				Status:         new("ACTIVE"),
				DesiredCount:   1,
				RunningCount:   1,
				PendingCount:   0,
				LaunchType:     types.LaunchTypeEc2,
				TaskDefinition: new("arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"), // link
				DeploymentConfiguration: &types.DeploymentConfiguration{
					DeploymentCircuitBreaker: &types.DeploymentCircuitBreaker{
						Enable:   false,
						Rollback: false,
					},
					MaximumPercent:        new(int32(200)),
					MinimumHealthyPercent: new(int32(100)),
					Alarms: &types.DeploymentAlarms{
						AlarmNames: []string{
							"foo",
						},
						Enable:   true,
						Rollback: true,
					},
				},
				Deployments: []types.Deployment{
					{
						Id:                 new("ecs-svc/6893472562508357546"),
						Status:             new("PRIMARY"),
						TaskDefinition:     new("arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"), // link
						DesiredCount:       1,
						PendingCount:       0,
						RunningCount:       1,
						FailedTasks:        0,
						CreatedAt:          new(time.Now()),
						UpdatedAt:          new(time.Now()),
						LaunchType:         types.LaunchTypeEc2,
						RolloutState:       types.DeploymentRolloutStateCompleted,
						RolloutStateReason: new("ECS deployment ecs-svc/6893472562508357546 completed."),
						CapacityProviderStrategy: []types.CapacityProviderStrategyItem{
							{
								CapacityProvider: new("provider"), // link
								Base:             10,
								Weight:           10,
							},
						},
						NetworkConfiguration: &types.NetworkConfiguration{
							AwsvpcConfiguration: &types.AwsVpcConfiguration{
								Subnets: []string{
									"subnet", // link
								},
								AssignPublicIp: types.AssignPublicIpEnabled,
								SecurityGroups: []string{
									"sg1", // link
								},
							},
						},
						PlatformFamily:  new("foo"),
						PlatformVersion: new("LATEST"),
						ServiceConnectConfiguration: &types.ServiceConnectConfiguration{
							Enabled: true,
							LogConfiguration: &types.LogConfiguration{
								LogDriver: types.LogDriverAwslogs,
								Options:   map[string]string{},
								SecretOptions: []types.Secret{
									{
										Name:      new("something"),
										ValueFrom: new("somewhere"),
									},
								},
							},
							Namespace: new("namespace"),
							Services: []types.ServiceConnectService{
								{
									PortName: new("http"),
									ClientAliases: []types.ServiceConnectClientAlias{
										{
											Port:    new(int32(80)),
											DnsName: new("www.foo.com"), // link
										},
									},
								},
							},
						},
						ServiceConnectResources: []types.ServiceConnectServiceResource{
							{
								DiscoveryArn:  new("arn:aws:service:region:account:layer:name:version"), // link
								DiscoveryName: new("name"),
							},
						},
					},
				},
				RoleArn: new("arn:aws:iam::052392120703:role/ecs-template-ECSServiceRole-1IL5CNMR1600J"),
				Events: []types.ServiceEvent{
					{
						Id:        new("a727ef2a-8a38-4746-905e-b529c952edee"),
						CreatedAt: new(time.Now()),
						Message:   new("(service ecs-template-service-i0mQKzkhDI2C) has reached a steady state."),
					},
					{
						Id:        new("69489991-f8ee-42a2-94f2-db8ffeda1ee7"),
						CreatedAt: new(time.Now()),
						Message:   new("(service ecs-template-service-i0mQKzkhDI2C) (deployment ecs-svc/6893472562508357546) deployment completed."),
					},
					{
						Id:        new("9ce65c4b-2993-477d-aa83-dbe98988f90b"),
						CreatedAt: new(time.Now()),
						Message:   new("(service ecs-template-service-i0mQKzkhDI2C) registered 1 targets in (target-group arn:aws:elasticloadbalancing:eu-west-1:052392120703:targetgroup/ECSTG/0c44b1cdb3437902)"),
					},
					{
						Id:        new("753e988a-9fb9-4907-b801-5f67369bc0de"),
						CreatedAt: new(time.Now()),
						Message:   new("(service ecs-template-service-i0mQKzkhDI2C) has started 1 tasks: (task 53074e0156204f30a3cea97e7bf32d31)."),
					},
					{
						Id:        new("deb2400b-a776-4ebe-8c97-f94feef2b780"),
						CreatedAt: new(time.Now()),
						Message:   new("(service ecs-template-service-i0mQKzkhDI2C) was unable to place a task because no container instance met all of its requirements. Reason: No Container Instances were found in your cluster. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."),
					},
				},
				CreatedAt: new(time.Now()),
				PlacementConstraints: []types.PlacementConstraint{
					{
						Expression: new("expression"),
						Type:       types.PlacementConstraintTypeDistinctInstance,
					},
				},
				PlacementStrategy: []types.PlacementStrategy{
					{
						Field: new("field"),
						Type:  types.PlacementStrategyTypeSpread,
					},
				},
				HealthCheckGracePeriodSeconds: new(int32(0)),
				SchedulingStrategy:            types.SchedulingStrategyReplica,
				DeploymentController: &types.DeploymentController{
					Type: types.DeploymentControllerTypeEcs,
				},
				CreatedBy:            new("arn:aws:iam::052392120703:role/aws-reserved/sso.amazonaws.com/eu-west-2/AWSReservedSSO_AWSAdministratorAccess_c1c3c9c54821c68a"),
				EnableECSManagedTags: false,
				PropagateTags:        types.PropagateTagsNone,
				EnableExecuteCommand: false,
				CapacityProviderStrategy: []types.CapacityProviderStrategyItem{
					{
						CapacityProvider: new("provider"),
						Base:             10,
						Weight:           10,
					},
				},
				NetworkConfiguration: &types.NetworkConfiguration{
					AwsvpcConfiguration: &types.AwsVpcConfiguration{
						Subnets: []string{
							"subnet2", // link
						},
						AssignPublicIp: types.AssignPublicIpEnabled,
						SecurityGroups: []string{
							"sg2", // link
						},
					},
				},
				PlatformFamily:  new("family"),
				PlatformVersion: new("LATEST"),
				Tags:            []types.Tag{},
				TaskSets: []types.TaskSet{
					// This seems to be able to return the *entire* task set,
					// which is redundant info. We should remove everything
					// other than the IDs
					{
						Id: new("id"), // link, then remove
					},
				},
			},
		},
	}, nil
}

func (t *ecsTestClient) ListServices(context.Context, *ecs.ListServicesInput, ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return &ecs.ListServicesOutput{
		ServiceArns: []string{
			"arn:aws:ecs:eu-west-1:052392120703:service/ecs-template-ECSCluster-8nS0WOLbs3nZ/ecs-template-service-i0mQKzkhDI2C",
		},
	}, nil
}

func TestServiceGetFunc(t *testing.T) {
	item, err := serviceGetFunc(context.Background(), &ecsTestClient{}, "foo", &ecs.DescribeServicesInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "ecs-cluster",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ecs:eu-west-1:052392120703:cluster/ecs-template-ECSCluster-8nS0WOLbs3nZ",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:elasticloadbalancing:eu-west-1:052392120703:targetgroup/ECSTG/0c44b1cdb3437902",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "servicediscovery-service",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type:name",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "ecs-task-definition",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "ecs-task-definition",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "ecs-capacity-provider",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "provider",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ecs-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg1",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "www.foo.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "servicediscovery-service",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:layer:name:version",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet2",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg2",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ecs-task-set",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewECSServiceAdapter(t *testing.T) {
	client, account, region := ecsGetAutoConfig(t)

	adapter := NewECSServiceAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
