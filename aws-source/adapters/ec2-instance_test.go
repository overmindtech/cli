package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
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
						AmiLaunchIndex:  adapterhelpers.PtrInt32(0),
						PublicIpAddress: adapterhelpers.PtrString("43.5.36.7"),
						ImageId:         adapterhelpers.PtrString("ami-04706e771f950937f"),
						InstanceId:      adapterhelpers.PtrString("i-04c7b2794f7bc3d6a"),
						IamInstanceProfile: &types.IamInstanceProfile{
							Arn: adapterhelpers.PtrString("arn:aws:iam::052392120703:instance-profile/test"),
							Id:  adapterhelpers.PtrString("AIDAJQEAZVQ7Y2EYQ2Z6Q"),
						},
						BootMode:                types.BootModeValuesLegacyBios,
						CurrentInstanceBootMode: types.InstanceBootModeValuesLegacyBios,
						ElasticGpuAssociations: []types.ElasticGpuAssociation{
							{
								ElasticGpuAssociationId:    adapterhelpers.PtrString("ega-0a1b2c3d4e5f6g7h8"),
								ElasticGpuAssociationState: adapterhelpers.PtrString("associated"),
								ElasticGpuAssociationTime:  adapterhelpers.PtrString("now"),
								ElasticGpuId:               adapterhelpers.PtrString("egp-0a1b2c3d4e5f6g7h8"),
							},
						},
						CapacityReservationId: adapterhelpers.PtrString("cr-0a1b2c3d4e5f6g7h8"),
						InstanceType:          types.InstanceTypeT2Micro,
						ElasticInferenceAcceleratorAssociations: []types.ElasticInferenceAcceleratorAssociation{
							{
								ElasticInferenceAcceleratorArn:              adapterhelpers.PtrString("arn:aws:elastic-inference:us-east-1:052392120703:accelerator/eia-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationId:    adapterhelpers.PtrString("eiaa-0a1b2c3d4e5f6g7h8"),
								ElasticInferenceAcceleratorAssociationState: adapterhelpers.PtrString("associated"),
								ElasticInferenceAcceleratorAssociationTime:  adapterhelpers.PtrTime(time.Now()),
							},
						},
						InstanceLifecycle: types.InstanceLifecycleTypeScheduled,
						Ipv6Address:       adapterhelpers.PtrString("2001:db8:3333:4444:5555:6666:7777:8888"),
						KeyName:           adapterhelpers.PtrString("dylan.ratcliffe"),
						KernelId:          adapterhelpers.PtrString("aki-0a1b2c3d4e5f6g7h8"),
						Licenses: []types.LicenseConfiguration{
							{
								LicenseConfigurationArn: adapterhelpers.PtrString("arn:aws:license-manager:us-east-1:052392120703:license-configuration:lic-0a1b2c3d4e5f6g7h8"),
							},
						},
						OutpostArn:            adapterhelpers.PtrString("arn:aws:outposts:us-east-1:052392120703:outpost/op-0a1b2c3d4e5f6g7h8"),
						Platform:              types.PlatformValuesWindows,
						RamdiskId:             adapterhelpers.PtrString("ari-0a1b2c3d4e5f6g7h8"),
						SpotInstanceRequestId: adapterhelpers.PtrString("sir-0a1b2c3d4e5f6g7h8"),
						SriovNetSupport:       adapterhelpers.PtrString("simple"),
						StateReason: &types.StateReason{
							Code:    adapterhelpers.PtrString("foo"),
							Message: adapterhelpers.PtrString("bar"),
						},
						TpmSupport: adapterhelpers.PtrString("foo"),
						LaunchTime: adapterhelpers.PtrTime(time.Now()),
						Monitoring: &types.Monitoring{
							State: types.MonitoringStateDisabled,
						},
						Placement: &types.Placement{
							AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"), // link
							GroupName:        adapterhelpers.PtrString(""),
							GroupId:          adapterhelpers.PtrString("groupId"),
							Tenancy:          types.TenancyDefault,
						},
						PrivateDnsName:   adapterhelpers.PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
						PrivateIpAddress: adapterhelpers.PtrString("172.31.95.79"),
						ProductCodes:     []types.ProductCode{},
						PublicDnsName:    adapterhelpers.PtrString(""),
						State: &types.InstanceState{
							Code: adapterhelpers.PtrInt32(16),
							Name: types.InstanceStateNameRunning,
						},
						StateTransitionReason: adapterhelpers.PtrString(""),
						SubnetId:              adapterhelpers.PtrString("subnet-0450a637af9984235"),
						VpcId:                 adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
						Architecture:          types.ArchitectureValuesX8664,
						BlockDeviceMappings: []types.InstanceBlockDeviceMapping{
							{
								DeviceName: adapterhelpers.PtrString("/dev/xvda"),
								Ebs: &types.EbsInstanceBlockDevice{
									AttachTime:          adapterhelpers.PtrTime(time.Now()),
									DeleteOnTermination: adapterhelpers.PtrBool(true),
									Status:              types.AttachmentStatusAttached,
									VolumeId:            adapterhelpers.PtrString("vol-06c7211d9e79a355e"),
								},
							},
						},
						ClientToken:  adapterhelpers.PtrString("eafad400-29e0-4b5c-a0fc-ef74c77659c4"),
						EbsOptimized: adapterhelpers.PtrBool(false),
						EnaSupport:   adapterhelpers.PtrBool(true),
						Hypervisor:   types.HypervisorTypeXen,
						NetworkInterfaces: []types.InstanceNetworkInterface{
							{
								Attachment: &types.InstanceNetworkInterfaceAttachment{
									AttachTime:          adapterhelpers.PtrTime(time.Now()),
									AttachmentId:        adapterhelpers.PtrString("eni-attach-02b19215d0dd9c7be"),
									DeleteOnTermination: adapterhelpers.PtrBool(true),
									DeviceIndex:         adapterhelpers.PtrInt32(0),
									Status:              types.AttachmentStatusAttached,
									NetworkCardIndex:    adapterhelpers.PtrInt32(0),
								},
								Description: adapterhelpers.PtrString(""),
								Groups: []types.GroupIdentifier{
									{
										GroupName: adapterhelpers.PtrString("default"),
										GroupId:   adapterhelpers.PtrString("sg-094e151c9fc5da181"),
									},
								},
								Ipv6Addresses:      []types.InstanceIpv6Address{},
								MacAddress:         adapterhelpers.PtrString("02:8c:61:38:6f:c2"),
								NetworkInterfaceId: adapterhelpers.PtrString("eni-09711a69e6d511358"),
								OwnerId:            adapterhelpers.PtrString("052392120703"),
								PrivateDnsName:     adapterhelpers.PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
								PrivateIpAddress:   adapterhelpers.PtrString("172.31.95.79"),
								PrivateIpAddresses: []types.InstancePrivateIpAddress{
									{
										Primary:          adapterhelpers.PtrBool(true),
										PrivateDnsName:   adapterhelpers.PtrString("ip-172-31-95-79.eu-west-2.compute.internal"),
										PrivateIpAddress: adapterhelpers.PtrString("172.31.95.79"),
									},
								},
								SourceDestCheck: adapterhelpers.PtrBool(true),
								Status:          types.NetworkInterfaceStatusInUse,
								SubnetId:        adapterhelpers.PtrString("subnet-0450a637af9984235"),
								VpcId:           adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
								InterfaceType:   adapterhelpers.PtrString("interface"),
							},
						},
						RootDeviceName: adapterhelpers.PtrString("/dev/xvda"),
						RootDeviceType: types.DeviceTypeEbs,
						SecurityGroups: []types.GroupIdentifier{
							{
								GroupName: adapterhelpers.PtrString("default"),
								GroupId:   adapterhelpers.PtrString("sg-094e151c9fc5da181"),
							},
						},
						SourceDestCheck: adapterhelpers.PtrBool(true),
						Tags: []types.Tag{
							{
								Key:   adapterhelpers.PtrString("Name"),
								Value: adapterhelpers.PtrString("test"),
							},
						},
						VirtualizationType: types.VirtualizationTypeHvm,
						CpuOptions: &types.CpuOptions{
							CoreCount:      adapterhelpers.PtrInt32(1),
							ThreadsPerCore: adapterhelpers.PtrInt32(1),
						},
						CapacityReservationSpecification: &types.CapacityReservationSpecificationResponse{
							CapacityReservationPreference: types.CapacityReservationPreferenceOpen,
						},
						HibernationOptions: &types.HibernationOptions{
							Configured: adapterhelpers.PtrBool(false),
						},
						MetadataOptions: &types.InstanceMetadataOptionsResponse{
							State:                   types.InstanceMetadataOptionsStateApplied,
							HttpTokens:              types.HttpTokensStateOptional,
							HttpPutResponseHopLimit: adapterhelpers.PtrInt32(1),
							HttpEndpoint:            types.InstanceMetadataEndpointStateEnabled,
							HttpProtocolIpv6:        types.InstanceMetadataProtocolStateDisabled,
							InstanceMetadataTags:    types.InstanceMetadataTagsStateDisabled,
						},
						EnclaveOptions: &types.EnclaveOptions{
							Enabled: adapterhelpers.PtrBool(false),
						},
						PlatformDetails:          adapterhelpers.PtrString("Linux/UNIX"),
						UsageOperation:           adapterhelpers.PtrString("RunInstances"),
						UsageOperationUpdateTime: adapterhelpers.PtrTime(time.Now()),
						PrivateDnsNameOptions: &types.PrivateDnsNameOptionsResponse{
							HostnameType:                    types.HostnameTypeIpName,
							EnableResourceNameDnsARecord:    adapterhelpers.PtrBool(true),
							EnableResourceNameDnsAAAARecord: adapterhelpers.PtrBool(false),
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
	tests := adapterhelpers.QueryTests{
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

	adapter := NewEC2InstanceAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
