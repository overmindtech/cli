package adapters

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/overmindtech/cli/sdp-go"
	"testing"
	"time"
)

func (c testNetworkFirewallClient) DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error) {
	now := time.Now()
	return &networkfirewall.DescribeFirewallPolicyOutput{
		FirewallPolicyResponse: &types.FirewallPolicyResponse{
			FirewallPolicyArn:             PtrString("arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3"),
			FirewallPolicyId:              PtrString("test"),
			FirewallPolicyName:            PtrString("test"),
			ConsumedStatefulRuleCapacity:  PtrInt32(1),
			ConsumedStatelessRuleCapacity: PtrInt32(1),
			Description:                   PtrString("test"),
			EncryptionConfiguration: &types.EncryptionConfiguration{
				Type:  types.EncryptionTypeAwsOwnedKmsKey,
				KeyId: PtrString("arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"), // link (this can be an ARN or ID)
			},
			FirewallPolicyStatus: types.ResourceStatusActive, // health
			LastModifiedTime:     &now,
			NumberOfAssociations: PtrInt32(1),
			Tags: []types.Tag{
				{
					Key:   PtrString("test"),
					Value: PtrString("test"),
				},
			},
		},
		FirewallPolicy: &types.FirewallPolicy{
			StatelessDefaultActions:         []string{},
			StatelessFragmentDefaultActions: []string{},
			PolicyVariables: &types.PolicyVariables{
				RuleVariables: map[string]types.IPSet{
					"test": {
						Definition: []string{},
					},
				},
			},
			StatefulDefaultActions: []string{},
			StatefulEngineOptions: &types.StatefulEngineOptions{
				RuleOrder:             types.RuleOrderDefaultActionOrder,
				StreamExceptionPolicy: types.StreamExceptionPolicyContinue,
			},
			StatefulRuleGroupReferences: []types.StatefulRuleGroupReference{
				{
					ResourceArn: PtrString("arn:aws:network-firewall:us-east-1:123456789012:stateful-rulegroup/aws-network-firewall-DefaultStatefulRuleGroup-1J3Z3W2ZQXV3"), // link
					Override: &types.StatefulRuleGroupOverride{
						Action: types.OverrideActionDropToAlert,
					},
					Priority: PtrInt32(1),
				},
			},
			StatelessCustomActions: []types.CustomAction{
				{
					ActionDefinition: &types.ActionDefinition{
						PublishMetricAction: &types.PublishMetricAction{
							Dimensions: []types.Dimension{},
						},
					},
					ActionName: PtrString("test"),
				},
			},
			StatelessRuleGroupReferences: []types.StatelessRuleGroupReference{
				{
					Priority:    PtrInt32(1),
					ResourceArn: PtrString("arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3"), // link
				},
			},
			TLSInspectionConfigurationArn: PtrString("arn:aws:network-firewall:us-east-1:123456789012:tls-inspection-configuration/aws-network-firewall-DefaultTlsInspectionConfiguration-1J3Z3W2ZQXV3"), // link
		},
	}, nil
}

func (c testNetworkFirewallClient) ListFirewallPolicies(context.Context, *networkfirewall.ListFirewallPoliciesInput, ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &networkfirewall.ListFirewallPoliciesOutput{
		FirewallPolicies: []types.FirewallPolicyMetadata{
			{
				Arn: PtrString("arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3"),
			},
		},
	}, nil
}

func TestFirewallPolicyGetFunc(t *testing.T) {
	item, err := firewallPolicyGetFunc(context.Background(), testNetworkFirewallClient{}, "test", &networkfirewall.DescribeFirewallPolicyInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "network-firewall-rule-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:network-firewall:us-east-1:123456789012:stateful-rulegroup/aws-network-firewall-DefaultStatefulRuleGroup-1J3Z3W2ZQXV3",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "network-firewall-rule-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/aws-network-firewall-DefaultStatelessRuleGroup-1J3Z3W2ZQXV3",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "network-firewall-tls-inspection-configuration",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:network-firewall:us-east-1:123456789012:tls-inspection-configuration/aws-network-firewall-DefaultTlsInspectionConfiguration-1J3Z3W2ZQXV3",
			ExpectedScope:  "123456789012.us-east-1",
		},
	}

	tests.Execute(t, item)
}
