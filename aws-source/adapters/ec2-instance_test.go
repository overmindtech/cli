package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestInstanceInputMapperGet(t *testing.T) {
	input, err := instanceInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.InstanceIds) != 1 {
		t.Fatalf("expected 1 instance ID, got %v", len(input.InstanceIds))
	}

	if input.InstanceIds[0] != "bar" {
		t.Errorf("expected instance ID to be bar, got %v", input.InstanceIds[0])
	}
}

func TestInstanceInputMapperList(t *testing.T) {
	input, err := instanceInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.InstanceIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestInstanceOutputMapper(t *testing.T) {
	output := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						AmiLaunchIndex:  new(int32(0)),
						PublicIpAddress: new("43.5.36.7"),
						ImageId:         new("ami-04706e771f950937f"),
						InstanceId:      new("i-04c7b2794f7bc3d6a"),
						IamInstanceProfile: &types.IamInstanceProfile{
							Arn: new("arn:aws:iam::052392120703:instance-profile/test"),
							Id:  new("AIDAJQEAZVQ7Y2EYQ2Z6Q"),
						},
						BootMode:                types.BootModeValuesLegacyBios,
						CurrentInstanceBootMode: types.InstanceBootModeValuesLegacyBios,
						ElasticGpuAssociations: []types.ElasticGpuAssociation{
							{
								ElasticGpuAssociationId:    new("ega-0a1b2c3d4e5f6g7h8"),
								ElasticGpuAssociationState: new("associated"),
								ElasticGpuAssociationTime:  new("now"),
								ElasticGpuId:               new("egp-0a1b2c3d4e5f6g7h8"),
							},
						},
						CapacityReservationId: new("cr-0a1b2c3d4e5f6g7h8"),
						InstanceType:          types.InstanceTypeT2Micro,
						ElasticInferenceAcceleratorAssociations: []types.ElasticInferenceAcceleratorAssociation{
							{
								ElasticInferenceAcceleratorArn:              new("arn:aws:elastic-inference:us-east-1:052392120703:accelerator/eia-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationId:    new("eiaa-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationState: new("associated"),
								ElasticInferenceAcceleratorAssociationTime:  new(time.Now()),
							},
						},
						InstanceLifecycle: types.InstanceLifecycleTypeScheduled,
						Ipv6Address:       new("2001:db8:3333:4444:5555:6666:7777:8888"),
						KeyName:           new("dylan.ratcliffe"),
						KernelId:          new("aki-0a1b2c3d4e5f6g7h8"),
						Licenses: []types.LicenseConfiguration{
							{
								LicenseConfigurationArn: new("arn:aws:license-manager:us-east-1:052392120703:license-configuration:lic-0a1b2c3d4e5f6g7h8"),
							},
						},
						OutpostArn:            new("arn:aws:outposts:us-east-1:052392120703:outpost/op-0a1b2c3d4e5f6g7h8"),
						Platform:              types.PlatformValuesWindows,
						RamdiskId:             new("ari-0a1b2c3d4e5f6g7h8"),
						SpotInstanceRequestId: new("sir-0a1b2c3d4e5f6g7h8"),
						SriovNetSupport:       new("simple"),
						StateReason: &types.StateReason{
							Code:    new("foo"),
							Message: new("bar"),
						},
						TpmSupport: new("foo"),
						LaunchTime: new(time.Now()),
						Monitoring: &types.Monitoring{
							State: types.MonitoringStateDisabled,
						},
						Placement: &types.Placement{
							AvailabilityZone: new("eu-west-2c"), // link
							GroupName:        new(""),
							GroupId:          new("groupId"),
							Tenancy:          types.TenancyDefault,
						},
						PrivateDnsName:   new("ip-172-31-95-79.eu-west-2.compute.internal"),
						PrivateIpAddress: new("172.31.95.79"),
						ProductCodes:     []types.ProductCode{},
						PublicDnsName:    new(""),
						State: &types.InstanceState{
							Code: new(int32(16)),
							Name: types.InstanceStateNameRunning,
						},
						StateTransitionReason: new(""),
						SubnetId:              new("subnet-0450a637af9984235"),
						VpcId:                 new("vpc-0d7892e00e573e701"),
						Architecture:          types.ArchitectureValuesX8664,
						BlockDeviceMappings: []types.InstanceBlockDeviceMapping{
							{
								DeviceName: new("/dev/xvda"),
								Ebs: &types.EbsInstanceBlockDevice{
									AttachTime:          new(time.Now()),
									DeleteOnTermination: new(true),
									Status:              types.AttachmentStatusAttached,
									VolumeId:            new("vol-06c7211d9e79a355e"),
								},
							},
						},
						ClientToken:  new("eafad400-29e0-4b5c-a0fc-ef74c77659c4"),
						EbsOptimized: new(false),
						EnaSupport:   new(true),
						Hypervisor:   types.HypervisorTypeXen,
						NetworkInterfaces: []types.InstanceNetworkInterface{
							{
								Attachment: &types.InstanceNetworkInterfaceAttachment{
									AttachTime:          new(time.Now()),
									AttachmentId:        new("eni-attach-02b19215d0dd9c7be"),
									DeleteOnTermination: new(true),
									DeviceIndex:         new(int32(0)),
									Status:              types.AttachmentStatusAttached,
									NetworkCardIndex:    new(int32(0)),
								},
								Description: new(""),
								Groups: []types.GroupIdentifier{
									{
										GroupName: new("default"),
										GroupId:   new("sg-094e151c9fc5da181"),
									},
								},
								Ipv6Addresses:      []types.InstanceIpv6Address{},
								MacAddress:         new("02:8c:61:38:6f:c2"),
								NetworkInterfaceId: new("eni-09711a69e6d511358"),
								OwnerId:            new("052392120703"),
								PrivateDnsName:     new("ip-172-31-95-79.eu-west-2.compute.internal"),
								PrivateIpAddress:   new("172.31.95.79"),
								PrivateIpAddresses: []types.InstancePrivateIpAddress{
									{
										Primary:          new(true),
										PrivateDnsName:   new("ip-172-31-95-79.eu-west-2.compute.internal"),
										PrivateIpAddress: new("172.31.95.79"),
									},
								},
								SourceDestCheck: new(true),
								Status:          types.NetworkInterfaceStatusInUse,
								SubnetId:        new("subnet-0450a637af9984235"),
								VpcId:           new("vpc-0d7892e00e573e701"),
								InterfaceType:   new("interface"),
							},
						},
						RootDeviceName: new("/dev/xvda"),
						RootDeviceType: types.DeviceTypeEbs,
						SecurityGroups: []types.GroupIdentifier{
							{
								GroupName: new("default"),
								GroupId:   new("sg-094e151c9fc5da181"),
							},
						},
						SourceDestCheck: new(true),
						Tags: []types.Tag{
							{
								Key:   new("Name"),
								Value: new("test"),
							},
						},
						VirtualizationType: types.VirtualizationTypeHvm,
						CpuOptions: &types.CpuOptions{
							CoreCount:      new(int32(1)),
							ThreadsPerCore: new(int32(1)),
						},
						CapacityReservationSpecification: &types.CapacityReservationSpecificationResponse{
							CapacityReservationPreference: types.CapacityReservationPreferenceOpen,
						},
						HibernationOptions: &types.HibernationOptions{
							Configured: new(false),
						},
						MetadataOptions: &types.InstanceMetadataOptionsResponse{
							State:                   types.InstanceMetadataOptionsStateApplied,
							HttpTokens:              types.HttpTokensStateOptional,
							HttpPutResponseHopLimit: new(int32(1)),
							HttpEndpoint:            types.InstanceMetadataEndpointStateEnabled,
							HttpProtocolIpv6:        types.InstanceMetadataProtocolStateDisabled,
							InstanceMetadataTags:    types.InstanceMetadataTagsStateDisabled,
						},
						EnclaveOptions: &types.EnclaveOptions{
							Enabled: new(false),
						},
						PlatformDetails:          new("Linux/UNIX"),
						UsageOperation:           new("RunInstances"),
						UsageOperationUpdateTime: new(time.Now()),
						PrivateDnsNameOptions: &types.PrivateDnsNameOptionsResponse{
							HostnameType:                    types.HostnameTypeIpName,
							EnableResourceNameDnsARecord:    new(true),
							EnableResourceNameDnsAAAARecord: new(false),
						},
						MaintenanceOptions: &types.InstanceMaintenanceOptions{
							AutoRecovery: types.InstanceAutoRecoveryStateDefault,
						},
					},
				},
			},
		},
	}

	items, err := instanceOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := QueryTests{
		{
			ExpectedType:   "ec2-image",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "ami-04706e771f950937f",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "172.31.95.79",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet-0450a637af9984235",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0d7892e00e573e701",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "iam-instance-profile",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::052392120703:instance-profile/test",
			ExpectedScope:  "052392120703",
		},
		{
			ExpectedType:   "ec2-capacity-reservation",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cr-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-elastic-gpu",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "egp-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "elastic-inference-accelerator",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:elastic-inference:us-east-1:052392120703:accelerator/eia-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  "052392120703.us-east-1",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "2001:db8:3333:4444:5555:6666:7777:8888",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "license-manager-license-configuration",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:license-manager:us-east-1:052392120703:license-configuration:lic-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  "052392120703.us-east-1",
		},
		{
			ExpectedType:   "outposts-outpost",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:outposts:us-east-1:052392120703:outpost/op-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  "052392120703.us-east-1",
		},
		{
			ExpectedType:   "ec2-spot-instance-request",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sir-0a1b2c3d4e5f6g7h8",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "43.5.36.7",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg-094e151c9fc5da181",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-instance-status",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-04c7b2794f7bc3d6a",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-volume",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vol-06c7211d9e79a355e",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-placement-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "groupId",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2InstanceAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2InstanceAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
