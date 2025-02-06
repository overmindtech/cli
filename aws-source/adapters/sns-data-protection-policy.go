package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type dataProtectionPolicyClient interface {
	GetDataProtectionPolicy(ctx context.Context, params *sns.GetDataProtectionPolicyInput, optFns ...func(*sns.Options)) (*sns.GetDataProtectionPolicyOutput, error)
}

func getDataProtectionPolicyFunc(ctx context.Context, client dataProtectionPolicyClient, scope string, input *sns.GetDataProtectionPolicyInput) (*sdp.Item, error) {
	output, err := client.GetDataProtectionPolicy(ctx, input)
	if err != nil {
		return nil, err
	}

	if output.DataProtectionPolicy == nil || *output.DataProtectionPolicy == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "get data protection policy response was nil/empty",
		}
	}

	// ResourceArn is the topic ARN that the policy is associated with
	attr := map[string]interface{}{
		"TopicArn": *input.ResourceArn,
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(attr)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "sns-data-protection-policy",
		UniqueAttribute: "TopicArn",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "sns-topic",
			Method: sdp.QueryMethod_GET,
			Query:  *input.ResourceArn,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Deleting the topic will delete the inline policy
			In: true,
			// Changing policy will affect the topic:
			//	a new statement denying credit card numbers will make the topic stop delivering messages
			//	containing credit card numbers
			Out: true,
		},
	})

	return item, nil
}

func NewSNSDataProtectionPolicyAdapter(client dataProtectionPolicyClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[any, any, *sns.GetDataProtectionPolicyInput, *sns.GetDataProtectionPolicyOutput, dataProtectionPolicyClient, *sns.Options] {
	return &adapterhelpers.AlwaysGetAdapter[any, any, *sns.GetDataProtectionPolicyInput, *sns.GetDataProtectionPolicyOutput, dataProtectionPolicyClient, *sns.Options]{
		ItemType:        "sns-data-protection-policy",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		DisableList:     true,
		AdapterMetadata: dataProtectionPolicyAdapterMetadata,
		GetInputMapper: func(scope, query string) *sns.GetDataProtectionPolicyInput {
			return &sns.GetDataProtectionPolicyInput{
				ResourceArn: &query,
			}
		},
		GetFunc: getDataProtectionPolicyFunc,
	}
}

var dataProtectionPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "sns-data-protection-policy",
	DescriptiveName: "SNS Data Protection Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an SNS data protection policy by associated topic ARN",
		SearchDescription: "Search SNS data protection policies by its ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_sns_topic_data_protection_policy.arn"},
	},
	PotentialLinks: []string{"sns-topic"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
