package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"

	log "github.com/sirupsen/logrus"
)

type keyPolicyClient interface {
	GetKeyPolicy(ctx context.Context, params *kms.GetKeyPolicyInput, optFns ...func(*kms.Options)) (*kms.GetKeyPolicyOutput, error)
	ListKeyPolicies(ctx context.Context, params *kms.ListKeyPoliciesInput, optFns ...func(*kms.Options)) (*kms.ListKeyPoliciesOutput, error)
}

func getKeyPolicyFunc(ctx context.Context, client keyPolicyClient, scope string, input *kms.GetKeyPolicyInput) (*sdp.Item, error) {
	output, err := client.GetKeyPolicy(ctx, input)
	if err != nil {
		return nil, err
	}

	if output.Policy == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "get key policy response was nil",
		}
	}

	type keyParsedPolicy struct {
		*kms.GetKeyPolicyOutput
		PolicyDocument *policy.Policy
	}

	parsedPolicy := keyParsedPolicy{
		GetKeyPolicyOutput: output,
	}

	parsedPolicy.PolicyDocument, err = ParsePolicyDocument(*output.Policy)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"input": input,
			"scope": scope,
		}).Error("Error parsing policy document")

		return nil, nil //nolint:nilerr
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(parsedPolicy)
	if err != nil {
		return nil, err
	}

	err = attributes.Set("KeyId", *input.KeyId)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "kms-key-policy",
		UniqueAttribute: "KeyId",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "kms-key",
			Method: sdp.QueryMethod_GET,
			Query:  *input.KeyId,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// These are tightly coupled
			In:  true,
			Out: true,
		},
	})

	return item, nil
}

func NewKMSKeyPolicyAdapter(client keyPolicyClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*kms.ListKeyPoliciesInput, *kms.ListKeyPoliciesOutput, *kms.GetKeyPolicyInput, *kms.GetKeyPolicyOutput, keyPolicyClient, *kms.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*kms.ListKeyPoliciesInput, *kms.ListKeyPoliciesOutput, *kms.GetKeyPolicyInput, *kms.GetKeyPolicyOutput, keyPolicyClient, *kms.Options]{
		ItemType:        "kms-key-policy",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		DisableList:     true, // This adapter only supports listing by Key ID
		AdapterMetadata: keyPolicyAdapterMetadata,
		SearchInputMapper: func(scope, query string) (*kms.ListKeyPoliciesInput, error) {
			return &kms.ListKeyPoliciesInput{
				KeyId: &query,
			}, nil
		},
		GetInputMapper: func(scope, query string) *kms.GetKeyPolicyInput {
			return &kms.GetKeyPolicyInput{
				KeyId: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client keyPolicyClient, input *kms.ListKeyPoliciesInput) adapterhelpers.Paginator[*kms.ListKeyPoliciesOutput, *kms.Options] {
			return kms.NewListKeyPoliciesPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *kms.ListKeyPoliciesOutput, input *kms.ListKeyPoliciesInput) ([]*kms.GetKeyPolicyInput, error) {
			var inputs []*kms.GetKeyPolicyInput
			for _, policyName := range output.PolicyNames {
				inputs = append(inputs, &kms.GetKeyPolicyInput{
					KeyId:      input.KeyId,
					PolicyName: &policyName,
				})
			}
			return inputs, nil
		},
		GetFunc: getKeyPolicyFunc,
	}
}

var keyPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "kms-key-policy",
	DescriptiveName: "KMS Key Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a KMS key policy by its Key ID",
		SearchDescription: "Search KMS key policies by Key ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_kms_key_policy.key_id"},
	},
	PotentialLinks: []string{"kms-key"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
