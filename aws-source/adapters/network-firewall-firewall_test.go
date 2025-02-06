package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (c testNetworkFirewallClient) DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error) {
	return &networkfirewall.DescribeFirewallOutput{
		Firewall: &types.Firewall{
			FirewallId:        adapterhelpers.PtrString("test"),
			FirewallPolicyArn: adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3"), // link
			SubnetMappings: []types.SubnetMapping{
				{
					SubnetId:      adapterhelpers.PtrString("subnet-12345678901234567"), // link
					IPAddressType: types.IPAddressTypeIpv4,
				},
			},
			VpcId:            adapterhelpers.PtrString("vpc-12345678901234567"), // link
			DeleteProtection: false,
			Description:      adapterhelpers.PtrString("test"),
			EncryptionConfiguration: &types.EncryptionConfiguration{
				Type:  types.EncryptionTypeAwsOwnedKmsKey,
				KeyId: adapterhelpers.PtrString("arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"), // link (this can be an ARN or ID)
			},
			FirewallArn:                    adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:firewall/aws-network-firewall-DefaultFirewall-1J3Z3W2ZQXV3"),
			FirewallName:                   adapterhelpers.PtrString("test"),
			FirewallPolicyChangeProtection: false,
			SubnetChangeProtection:         false,
			Tags: []types.Tag{
				{
					Key:   adapterhelpers.PtrString("test"),
					Value: adapterhelpers.PtrString("test"),
				},
			},
		},
		FirewallStatus: &types.FirewallStatus{
			ConfigurationSyncStateSummary: types.ConfigurationSyncStateInSync,
			Status:                        types.FirewallStatusValueDeleting,
			CapacityUsageSummary: &types.CapacityUsageSummary{
				CIDRs: &types.CIDRSummary{
					AvailableCIDRCount: adapterhelpers.PtrInt32(1),
					IPSetReferences: map[string]types.IPSetMetadata{
						"test": {
							ResolvedCIDRCount: adapterhelpers.PtrInt32(1),
						},
					},
					UtilizedCIDRCount: adapterhelpers.PtrInt32(1),
				},
			},
			SyncStates: map[string]types.SyncState{
				"test": {
					Attachment: &types.Attachment{
						EndpointId:    adapterhelpers.PtrString("test"),
						Status:        types.AttachmentStatusCreating,
						StatusMessage: adapterhelpers.PtrString("test"),
						SubnetId:      adapterhelpers.PtrString("test"), // link,
					},
				},
			},
		},
	}, nil
}

func (c testNetworkFirewallClient) DescribeLoggingConfiguration(ctx context.Context, params *networkfirewall.DescribeLoggingConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeLoggingConfigurationOutput, error) {
	return &networkfirewall.DescribeLoggingConfigurationOutput{
		FirewallArn: adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:firewall/aws-network-firewall-DefaultFirewall-1J3Z3W2ZQXV3"),
		LoggingConfiguration: &types.LoggingConfiguration{
			LogDestinationConfigs: []types.LogDestinationConfig{
				{
					LogDestination: map[string]string{
						"bucketName": "DOC-EXAMPLE-BUCKET", // link
						"prefix":     "alerts",
					},
					LogDestinationType: types.LogDestinationTypeS3,
					LogType:            types.LogTypeAlert,
				},
				{
					LogDestinationType: types.LogDestinationTypeCloudwatchLogs,
					LogDestination: map[string]string{
						"logGroup": "alert-log-group", // link
					},
					LogType: types.LogTypeAlert,
				},
				{
					LogDestinationType: types.LogDestinationTypeKinesisDataFirehose,
					LogDestination: map[string]string{
						"deliveryStream": "alert-delivery-stream", // link
					},
					LogType: types.LogTypeAlert,
				},
			},
		},
	}, nil
}

func (c testNetworkFirewallClient) DescribeResourcePolicy(ctx context.Context, params *networkfirewall.DescribeResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeResourcePolicyOutput, error) {
	return &networkfirewall.DescribeResourcePolicyOutput{
		Policy: adapterhelpers.PtrString("test"), // link
	}, nil
}

func (c testNetworkFirewallClient) ListFirewalls(context.Context, *networkfirewall.ListFirewallsInput, ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallsOutput, error) {
	return &networkfirewall.ListFirewallsOutput{
		Firewalls: []types.FirewallMetadata{
			{
				FirewallArn: adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:firewall/aws-network-firewall-DefaultFirewall-1J3Z3W2ZQXV3"),
			},
		},
	}, nil
}

func TestFirewallGetFunc(t *testing.T) {
	item, err := firewallGetFunc(context.Background(), testNetworkFirewallClient{}, "test", &networkfirewall.DescribeFirewallInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet-12345678901234567",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "network-firewall-firewall-policy",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-12345678901234567",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "logs-log-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "alert-log-group",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "DOC-EXAMPLE-BUCKET",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "firehose-delivery-stream",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "alert-delivery-stream",
			ExpectedScope:  "test",
		},
	}

	tests.Execute(t, item)
}
