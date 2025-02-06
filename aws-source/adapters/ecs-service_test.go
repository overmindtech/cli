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

func (t *ecsTestClient) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return &ecs.DescribeServicesOutput{
		Failures: []types.Failure{},
		Services: []types.Service{
			{
				ServiceArn:  adapterhelpers.PtrString("arn:aws:ecs:eu-west-1:052392120703:service/ecs-template-ECSCluster-8nS0WOLbs3nZ/ecs-template-service-i0mQKzkhDI2C"),
				ServiceName: adapterhelpers.PtrString("ecs-template-service-i0mQKzkhDI2C"),
				ClusterArn:  adapterhelpers.PtrString("arn:aws:ecs:eu-west-1:052392120703:cluster/ecs-template-ECSCluster-8nS0WOLbs3nZ"), // link
				LoadBalancers: []types.LoadBalancer{
					{
						TargetGroupArn: adapterhelpers.PtrString("arn:aws:elasticloadbalancing:eu-west-1:052392120703:targetgroup/ECSTG/0c44b1cdb3437902"), // link
						ContainerName:  adapterhelpers.PtrString("simple-app"),
						ContainerPort:  adapterhelpers.PtrInt32(80),
					},
				},
				ServiceRegistries: []types.ServiceRegistry{
					{
						ContainerName: adapterhelpers.PtrString("name"),
						ContainerPort: adapterhelpers.PtrInt32(80),
						Port:          adapterhelpers.PtrInt32(80),
						RegistryArn:   adapterhelpers.PtrString("arn:aws:service:region:account:type:name"), // link
					},
				},
				Status:         adapterhelpers.PtrString("ACTIVE"),
				DesiredCount:   1,
				RunningCount:   1,
				PendingCount:   0,
				LaunchType:     types.LaunchTypeEc2,
				TaskDefinition: adapterhelpers.PtrString("arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"), // link
				DeploymentConfiguration: &types.DeploymentConfiguration{
					DeploymentCircuitBreaker: &types.DeploymentCircuitBreaker{
						Enable:   false,
						Rollback: false,
					},
					MaximumPercent:        adapterhelpers.PtrInt32(200),
					MinimumHealthyPercent: adapterhelpers.PtrInt32(100),
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
						Id:                 adapterhelpers.PtrString("ecs-svc/6893472562508357546"),
						Status:             adapterhelpers.PtrString("PRIMARY"),
						TaskDefinition:     adapterhelpers.PtrString("arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"), // link
						DesiredCount:       1,
						PendingCount:       0,
						RunningCount:       1,
						FailedTasks:        0,
						CreatedAt:          adapterhelpers.PtrTime(time.Now()),
						UpdatedAt:          adapterhelpers.PtrTime(time.Now()),
						LaunchType:         types.LaunchTypeEc2,
						RolloutState:       types.DeploymentRolloutStateCompleted,
						RolloutStateReason: adapterhelpers.PtrString("ECS deployment ecs-svc/6893472562508357546 completed."),
						CapacityProviderStrategy: []types.CapacityProviderStrategyItem{
							{
								CapacityProvider: adapterhelpers.PtrString("provider"), // link
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
						PlatformFamily:  adapterhelpers.PtrString("foo"),
						PlatformVersion: adapterhelpers.PtrString("LATEST"),
						ServiceConnectConfiguration: &types.ServiceConnectConfiguration{
							Enabled: true,
							LogConfiguration: &types.LogConfiguration{
								LogDriver: types.LogDriverAwslogs,
								Options:   map[string]string{},
								SecretOptions: []types.Secret{
									{
										Name:      adapterhelpers.PtrString("something"),
										ValueFrom: adapterhelpers.PtrString("somewhere"),
									},
								},
							},
							Namespace: adapterhelpers.PtrString("namespace"),
							Services: []types.ServiceConnectService{
								{
									PortName: adapterhelpers.PtrString("http"),
									ClientAliases: []types.ServiceConnectClientAlias{
										{
											Port:    adapterhelpers.PtrInt32(80),
											DnsName: adapterhelpers.PtrString("www.foo.com"), // link
										},
									},
								},
							},
						},
						ServiceConnectResources: []types.ServiceConnectServiceResource{
							{
								DiscoveryArn:  adapterhelpers.PtrString("arn:aws:service:region:account:layer:name:version"), // link
								DiscoveryName: adapterhelpers.PtrString("name"),
							},
						},
					},
				},
				RoleArn: adapterhelpers.PtrString("arn:aws:iam::052392120703:role/ecs-template-ECSServiceRole-1IL5CNMR1600J"),
				Events: []types.ServiceEvent{
					{
						Id:        adapterhelpers.PtrString("a727ef2a-8a38-4746-905e-b529c952edee"),
						CreatedAt: adapterhelpers.PtrTime(time.Now()),
						Message:   adapterhelpers.PtrString("(service ecs-template-service-i0mQKzkhDI2C) has reached a steady state."),
					},
					{
						Id:        adapterhelpers.PtrString("69489991-f8ee-42a2-94f2-db8ffeda1ee7"),
						CreatedAt: adapterhelpers.PtrTime(time.Now()),
						Message:   adapterhelpers.PtrString("(service ecs-template-service-i0mQKzkhDI2C) (deployment ecs-svc/6893472562508357546) deployment completed."),
					},
					{
						Id:        adapterhelpers.PtrString("9ce65c4b-2993-477d-aa83-dbe98988f90b"),
						CreatedAt: adapterhelpers.PtrTime(time.Now()),
						Message:   adapterhelpers.PtrString("(service ecs-template-service-i0mQKzkhDI2C) registered 1 targets in (target-group arn:aws:elasticloadbalancing:eu-west-1:052392120703:targetgroup/ECSTG/0c44b1cdb3437902)"),
					},
					{
						Id:        adapterhelpers.PtrString("753e988a-9fb9-4907-b801-5f67369bc0de"),
						CreatedAt: adapterhelpers.PtrTime(time.Now()),
						Message:   adapterhelpers.PtrString("(service ecs-template-service-i0mQKzkhDI2C) has started 1 tasks: (task 53074e0156204f30a3cea97e7bf32d31)."),
					},
					{
						Id:        adapterhelpers.PtrString("deb2400b-a776-4ebe-8c97-f94feef2b780"),
						CreatedAt: adapterhelpers.PtrTime(time.Now()),
						Message:   adapterhelpers.PtrString("(service ecs-template-service-i0mQKzkhDI2C) was unable to place a task because no container instance met all of its requirements. Reason: No Container Instances were found in your cluster. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."),
					},
				},
				CreatedAt: adapterhelpers.PtrTime(time.Now()),
				PlacementConstraints: []types.PlacementConstraint{
					{
						Expression: adapterhelpers.PtrString("expression"),
						Type:       types.PlacementConstraintTypeDistinctInstance,
					},
				},
				PlacementStrategy: []types.PlacementStrategy{
					{
						Field: adapterhelpers.PtrString("field"),
						Type:  types.PlacementStrategyTypeSpread,
					},
				},
				HealthCheckGracePeriodSeconds: adapterhelpers.PtrInt32(0),
				SchedulingStrategy:            types.SchedulingStrategyReplica,
				DeploymentController: &types.DeploymentController{
					Type: types.DeploymentControllerTypeEcs,
				},
				CreatedBy:            adapterhelpers.PtrString("arn:aws:iam::052392120703:role/aws-reserved/sso.amazonaws.com/eu-west-2/AWSReservedSSO_AWSAdministratorAccess_c1c3c9c54821c68a"),
				EnableECSManagedTags: false,
				PropagateTags:        types.PropagateTagsNone,
				EnableExecuteCommand: false,
				CapacityProviderStrategy: []types.CapacityProviderStrategyItem{
					{
						CapacityProvider: adapterhelpers.PtrString("provider"),
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
				PlatformFamily:  adapterhelpers.PtrString("family"),
				PlatformVersion: adapterhelpers.PtrString("LATEST"),
				Tags:            []types.Tag{},
				TaskSets: []types.TaskSet{
					// This seems to be able to return the *entire* task set,
					// which is redundant info. We should remove everything
					// other than the IDs
					{
						Id: adapterhelpers.PtrString("id"), // link, then remove
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

	tests := adapterhelpers.QueryTests{
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

	adapter := NewECSServiceAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
