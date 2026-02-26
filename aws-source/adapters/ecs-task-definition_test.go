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

func (t *ecsTestClient) DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	return &ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &types.TaskDefinition{
			TaskDefinitionArn: new("arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"),
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:   new("simple-app"),
					Image:  new("httpd:2.4"),
					Cpu:    10,
					Memory: new(int32(300)),
					Links:  []string{},
					PortMappings: []types.PortMapping{
						{
							ContainerPort: new(int32(80)),
							HostPort:      new(int32(0)),
							Protocol:      types.TransportProtocolTcp,
							AppProtocol:   types.ApplicationProtocolHttp,
						},
					},
					Essential:  new(true),
					EntryPoint: []string{},
					Command:    []string{},
					Environment: []types.KeyValuePair{
						{
							Name:  new("DATABASE_SERVER"),
							Value: new("database01.my-company.com"),
						},
					},
					EnvironmentFiles: []types.EnvironmentFile{},
					MountPoints: []types.MountPoint{
						{
							SourceVolume:  new("my-vol"),
							ContainerPath: new("/usr/local/apache2/htdocs"),
							ReadOnly:      new(false),
						},
					},
					VolumesFrom: []types.VolumeFrom{
						{
							SourceContainer: new("container"),
						},
					},
					Secrets: []types.Secret{
						{
							Name:      new("secrets-manager"),
							ValueFrom: new("arn:aws:secretsmanager:us-west-2:123456789012:secret:my-path/my-secret-name-1a2b3c"), // link
						},
						{
							Name:      new("ssm"),
							ValueFrom: new("arn:aws:ssm:us-east-2:123456789012:parameter/prod-123"), // link
						},
					},
					DnsServers:       []string{},
					DnsSearchDomains: []string{},
					ExtraHosts: []types.HostEntry{
						{
							Hostname:  new("host"),
							IpAddress: new("127.0.0.1"),
						},
					},
					DockerSecurityOptions: []string{},
					DockerLabels:          map[string]string{},
					Ulimits:               []types.Ulimit{},
					LogConfiguration: &types.LogConfiguration{
						LogDriver: types.LogDriverAwslogs,
						Options: map[string]string{
							"awslogs-group":         "ECSLogGroup-ecs-template",
							"awslogs-region":        "eu-west-1",
							"awslogs-stream-prefix": "ecs-demo-app",
						},
						SecretOptions: []types.Secret{
							{
								Name:      new("secrets-manager"),
								ValueFrom: new("arn:aws:secretsmanager:us-west-2:123456789012:secret:my-path/my-secret-name-1a2b3c"), // link
							},
							{
								Name:      new("ssm"),
								ValueFrom: new("arn:aws:ssm:us-east-2:123456789012:parameter/prod-123"), // link
							},
						},
					},
					SystemControls:    []types.SystemControl{},
					DependsOn:         []types.ContainerDependency{},
					DisableNetworking: new(false),
					FirelensConfiguration: &types.FirelensConfiguration{
						Type:    types.FirelensConfigurationTypeFluentd,
						Options: map[string]string{},
					},
					HealthCheck:            &types.HealthCheck{},
					Hostname:               new("hostname"),
					Interactive:            new(false),
					LinuxParameters:        &types.LinuxParameters{},
					MemoryReservation:      new(int32(100)),
					Privileged:             new(false),
					PseudoTerminal:         new(false),
					ReadonlyRootFilesystem: new(false),
					RepositoryCredentials:  &types.RepositoryCredentials{}, // Skipping the link here for now, if you need it, add it in a PR
					ResourceRequirements:   []types.ResourceRequirement{},
					StartTimeout:           new(int32(1)),
					StopTimeout:            new(int32(1)),
					User:                   new("foo"),
					WorkingDirectory:       new("/"),
				},
				{
					Name:      new("busybox"),
					Image:     new("busybox"),
					Cpu:       10,
					Memory:    new(int32(200)),
					Essential: new(false),
					EntryPoint: []string{
						"sh",
						"-c",
					},
					Command: []string{
						"/bin/sh -c \"while true; do echo '<html> <head> <title>Amazon ECS Sample App</title> <style>body {margin-top: 40px; background-color: #333;} </style> </head><body> <div style=color:white;text-align:center> <h1>Amazon ECS Sample App</h1> <h2>Congratulations!</h2> <p>Your application is now running on a container in Amazon ECS.</p>' > top; /bin/date > date ; echo '</div></body></html>' > bottom; cat top date bottom > /usr/local/apache2/htdocs/index.html ; sleep 1; done\"",
					},
					VolumesFrom: []types.VolumeFrom{
						{
							SourceContainer: new("simple-app"),
						},
					},
					DockerLabels: map[string]string{},
					LogConfiguration: &types.LogConfiguration{
						LogDriver: types.LogDriverAwslogs,
						Options: map[string]string{
							"awslogs-group":         "ECSLogGroup-ecs-template",
							"awslogs-region":        "eu-west-1",
							"awslogs-stream-prefix": "ecs-demo-app",
						},
					},
				},
			},
			Family:   new("ecs-template-ecs-demo-app"),
			Revision: 1,
			Volumes: []types.Volume{
				{
					Name: new("my-vol"),
					Host: &types.HostVolumeProperties{
						SourcePath: new("/"),
					},
				},
			},
			Status: types.TaskDefinitionStatusActive,
			RequiresAttributes: []types.Attribute{
				{
					Name: new("com.amazonaws.ecs.capability.logging-driver.awslogs"),
				},
				{
					Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.19"),
				},
				{
					Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.17"),
				},
				{
					Name: new("com.amazonaws.ecs.capability.docker-remote-api.1.18"),
				},
			},
			PlacementConstraints: []types.TaskDefinitionPlacementConstraint{},
			Compatibilities: []types.Compatibility{
				"EXTERNAL",
				"EC2",
			},
			RegisteredAt:   new(time.Now()),
			RegisteredBy:   new("arn:aws:sts::052392120703:assumed-role/AWSReservedSSO_AWSAdministratorAccess_c1c3c9c54821c68a/dylan@overmind.tech"),
			Cpu:            new("cpu"),
			DeregisteredAt: new(time.Now()),
			EphemeralStorage: &types.EphemeralStorage{
				SizeInGiB: 1,
			},
			ExecutionRoleArn:        new("arn:aws:iam:us-east-2:123456789012:role/foo"), // link
			InferenceAccelerators:   []types.InferenceAccelerator{},
			IpcMode:                 types.IpcModeHost,
			Memory:                  new("memory"),
			NetworkMode:             types.NetworkModeAwsvpc,
			PidMode:                 types.PidModeHost,
			ProxyConfiguration:      nil,
			RequiresCompatibilities: []types.Compatibility{},
			RuntimePlatform: &types.RuntimePlatform{
				CpuArchitecture:       types.CPUArchitectureX8664,
				OperatingSystemFamily: types.OSFamilyLinux,
			},
			TaskRoleArn: new("arn:aws:iam:us-east-2:123456789012:role/bar"), // link
		},
	}, nil
}

func (t *ecsTestClient) ListTaskDefinitions(context.Context, *ecs.ListTaskDefinitionsInput, ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error) {
	return &ecs.ListTaskDefinitionsOutput{
		TaskDefinitionArns: []string{
			"arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1",
		},
	}, nil
}

func TestTaskDefinitionGetFunc(t *testing.T) {
	item, err := taskDefinitionGetFunc(context.Background(), &ecsTestClient{}, "foo", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: new("ecs-template-ecs-demo-app:1"),
	})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "secretsmanager-secret",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-path/my-secret-name-1a2b3c",
			ExpectedScope:  "123456789012.us-west-2",
		},
		{
			ExpectedType:   "ssm-parameter",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ssm:us-east-2:123456789012:parameter/prod-123",
			ExpectedScope:  "123456789012.us-east-2",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam:us-east-2:123456789012:role/foo",
			ExpectedScope:  "123456789012.us-east-2",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam:us-east-2:123456789012:role/bar",
			ExpectedScope:  "123456789012.us-east-2",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "database01.my-company.com",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}

func TestNewECSTaskDefinitionAdapter(t *testing.T) {
	client, account, region := ecsGetAutoConfig(t)

	adapter := NewECSTaskDefinitionAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
