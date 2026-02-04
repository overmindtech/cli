package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
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
						AmiLaunchIndex:  PtrInt32(0),
						PublicIpAddress: PtrString("43.5.36.7"),
						ImageId:         PtrString("ami-04706e771f950937f"),
						InstanceId:      PtrString("i-04c7b2794f7bc3d6a"),
						IamInstanceProfile: &types.IamInstanceProfile{
							Arn: PtrString("arn:aws:iam::052392120703:instance-profile/test"),
							Id:  PtrString("AIDAJQEAZVQ7Y2EYQ2Z6Q"),
						},
						BootMode:                types.BootModeValuesLegacyBios,
						CurrentInstanceBootMode: types.InstanceBootModeValuesLegacyBios,
						ElasticGpuAssociations: []types.ElasticGpuAssociation{
							{
								ElasticGpuAssociationId:    PtrString("ega-0a1b2c3d4e5f6g7h8"),
								ElasticGpuAssociationState: PtrString("associated"),
								ElasticGpuAssociationTime:  PtrString("now"),
								ElasticGpuId:               PtrString("egp-0a1b2c3d4e5f6g7h8"),
							},
						},
						CapacityReservationId: PtrString("cr-0a1b2c3d4e5f6g7h8"),
						InstanceType:          types.InstanceTypeT2Micro,
						ElasticInferenceAcceleratorAssociations: []types.ElasticInferenceAcceleratorAssociation{
							{
								ElasticInferenceAcceleratorArn:              PtrString("arn:aws:elastic-inference:us-east-1:052392120703:accelerator/eia-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationId:    PtrString("eiaa-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationState: PtrString("associated"),
								ElasticInferenceAcceleratorAssociationTime:  PtrTime(time.Now()),
							},
						},
						InstanceLifecycle: types.InstanceLifecycleTypeScheduled,
						Ipv6Address:       PtrString("2001:db8:3333:4444:5555:6666:7777:8888"),
						KeyName:           PtrString("dylan.ratcliffe"),
						KernelId:          PtrString("aki-0a1b2c3d4e5f6g7h8"),
						Licenses: []types.LicenseConfiguration{
							{
								LicenseConfigurationArn: PtrString("arn:aws:license-manager:us-east-1:052392120703:license-configuration:lic-0a1b2c3d4e5f6g7h8"),
							},
						},
						OutpostArn:            PtrString("arn:aws:outposts:us-east-1:052392120703:outpost/op-0a1b2c3d4e5f6g7h8"),
						Platform:              types.PlatformValuesWindows,
						RamdiskId:             PtrString("ari-0a1b2c3d4e5f6g7h8"),
						SpotInstanceRequestId: PtrString("sir-0a1b2c3d4e5f6g7h8"),
						SriovNetSupport:       PtrString("simple"),
						StateReason: &types.StateReason{
							Code:    PtrString("foo"),
							Message: PtrString("bar"),
						},
						TpmSupport: PtrString("foo"),
						LaunchTime: PtrTime(time.Now()),
						Monitoring: &types.Monitoring{
							State: types.MonitoringStateDisabled,
						},
						Placement: &types.Placement{
							AvailabilityZone: PtrString("eu-west-2c"), // link
							GroupName:        PtrString(""),
							GroupId:          PtrString("groupId"),
							Tenancy:          types.TenancyDefault,
						},
						PrivateDnsName:   PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
						PrivateIpAddress: PtrString("172.31.95.79"),
						ProductCodes:     []types.ProductCode{},
						PublicDnsName:    PtrString(""),
						State: &types.InstanceState{
							Code: PtrInt32(16),
							Name: types.InstanceStateNameRunning,
						},
						StateTransitionReason: PtrString(""),
						SubnetId:              PtrString("subnet-0450a637af9984235"),
						VpcId:                 PtrString("vpc-0d7892e00e573e701"),
						Architecture:          types.ArchitectureValuesX8664,
						BlockDeviceMappings: []types.InstanceBlockDeviceMapping{
							{
								DeviceName: PtrString("/dev/xvda"),
								Ebs: &types.EbsInstanceBlockDevice{
									AttachTime:          PtrTime(time.Now()),
									DeleteOnTermination: PtrBool(true),
									Status:              types.AttachmentStatusAttached,
									VolumeId:            PtrString("vol-06c7211d9e79a355e"),
								},
							},
						},
						ClientToken:  PtrString("eafad400-29e0-4b5c-a0fc-ef74c77659c4"),
						EbsOptimized: PtrBool(false),
						EnaSupport:   PtrBool(true),
						Hypervisor:   types.HypervisorTypeXen,
						NetworkInterfaces: []types.InstanceNetworkInterface{
							{
								Attachment: &types.InstanceNetworkInterfaceAttachment{
									AttachTime:          PtrTime(time.Now()),
									AttachmentId:        PtrString("eni-attach-02b19215d0dd9c7be"),
									DeleteOnTermination: PtrBool(true),
									DeviceIndex:         PtrInt32(0),
									Status:              types.AttachmentStatusAttached,
									NetworkCardIndex:    PtrInt32(0),
								},
								Description: PtrString(""),
								Groups: []types.GroupIdentifier{
									{
										GroupName: PtrString("default"),
										GroupId:   PtrString("sg-094e151c9fc5da181"),
									},
								},
								Ipv6Addresses:      []types.InstanceIpv6Address{},
								MacAddress:         PtrString("02:8c:61:38:6f:c2"),
								NetworkInterfaceId: PtrString("eni-09711a69e6d511358"),
								OwnerId:            PtrString("052392120703"),
								PrivateDnsName:     PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
								PrivateIpAddress:   PtrString("172.31.95.79"),
								PrivateIpAddresses: []types.InstancePrivateIpAddress{
									{
										Primary:          PtrBool(true),
										PrivateDnsName:   PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
										PrivateIpAddress: PtrString("172.31.95.79"),
									},
								},
								SourceDestCheck: PtrBool(true),
								Status:          types.NetworkInterfaceStatusInUse,
								SubnetId:        PtrString("subnet-0450a637af9984235"),
								VpcId:           PtrString("vpc-0d7892e00e573e701"),
								InterfaceType:   PtrString("interface"),
							},
						},
						RootDeviceName: PtrString("/dev/xvda"),
						RootDeviceType: types.DeviceTypeEbs,
						SecurityGroups: []types.GroupIdentifier{
							{
								GroupName: PtrString("default"),
								GroupId:   PtrString("sg-094e151c9fc5da181"),
							},
						},
						SourceDestCheck: PtrBool(true),
						Tags: []types.Tag{
							{
								Key:   PtrString("Name"),
								Value: PtrString("test"),
							},
						},
						VirtualizationType: types.VirtualizationTypeHvm,
						CpuOptions: &types.CpuOptions{
							CoreCount:      PtrInt32(1),
							ThreadsPerCore: PtrInt32(1),
						},
						CapacityReservationSpecification: &types.CapacityReservationSpecificationResponse{
							CapacityReservationPreference: types.CapacityReservationPreferenceOpen,
						},
						HibernationOptions: &types.HibernationOptions{
							Configured: PtrBool(false),
						},
						MetadataOptions: &types.InstanceMetadataOptionsResponse{
							State:                   types.InstanceMetadataOptionsStateApplied,
							HttpTokens:              types.HttpTokensStateOptional,
							HttpPutResponseHopLimit: PtrInt32(1),
							HttpEndpoint:            types.InstanceMetadataEndpointStateEnabled,
							HttpProtocolIpv6:        types.InstanceMetadataProtocolStateDisabled,
							InstanceMetadataTags:    types.InstanceMetadataTagsStateDisabled,
						},
						EnclaveOptions: &types.EnclaveOptions{
							Enabled: PtrBool(false),
						},
						PlatformDetails:          PtrString("Linux/UNIX"),
						UsageOperation:           PtrString("RunInstances"),
						UsageOperationUpdateTime: PtrTime(time.Now()),
						PrivateDnsNameOptions: &types.PrivateDnsNameOptionsResponse{
							HostnameType:                    types.HostnameTypeIpName,
							EnableResourceNameDnsARecord:    PtrBool(true),
							EnableResourceNameDnsAAAARecord: PtrBool(false),
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
