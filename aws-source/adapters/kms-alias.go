package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/kms"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func aliasOutputMapper(_ context.Context, _ *kms.Client, scope string, _ *kms.ListAliasesInput, output *kms.ListAliasesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, alias := range output.Aliases {
		attributes, err := adapterhelpers.ToAttributesWithExclude(alias, "tags")
		if err != nil {
			return nil, err
		}

		// This should never happen.
		if alias.AliasName == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "aliasName is nil",
			}
		}

		// Ignore AWS managed keys, they are predefined and might not have a target key ID
		if strings.HasPrefix(*alias.AliasName, "alias/aws/") {
			// AWS managed keys
			continue
		}

		// This should never happen except for AWS managed keys.
		if alias.TargetKeyId == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "targetKeyId is nil",
			}
		}

		// The uniqueAttributeValue for this is the combination of the keyID and aliasName
		// i.e., "cf68415c-f4ae-48f2-87a7-3b52ce/alias/test-key"
		err = attributes.Set("UniqueName", fmt.Sprintf("%s/%s", *alias.TargetKeyId, *alias.AliasName))
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "kms-alias",
			UniqueAttribute: "UniqueName",
			Attributes:      attributes,
			Scope:           scope,
		}

		if alias.TargetKeyId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "kms-key",
					Method: sdp.QueryMethod_GET,
					Query:  *alias.TargetKeyId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					// Adding, deleting, or updating an alias can allow or deny permission to the KMS key.
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewKMSAliasAdapter(client *kms.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*kms.ListAliasesInput, *kms.ListAliasesOutput, *kms.Client, *kms.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*kms.ListAliasesInput, *kms.ListAliasesOutput, *kms.Client, *kms.Options]{
		ItemType:        "kms-alias",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: kmsAliasAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *kms.Client, input *kms.ListAliasesInput) (*kms.ListAliasesOutput, error) {
			return client.ListAliases(ctx, input)
		},
		InputMapperGet: func(_, query string) (*kms.ListAliasesInput, error) {
			// query must be in the format of: the keyID/aliasName
			// note that the aliasName will have a forward slash in it
			// i.e., "cf68415c-f4ae-48f2-87a7-3b52ce/alias/test-key"
			tmp := strings.Split(query, "/")
			if len(tmp) < 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the keyID/aliasName, but found: %s", query),
				}
			}

			return &kms.ListAliasesInput{
				KeyId: &tmp[0], // keyID
			}, nil
		},
		UseListForGet: true,
		InputMapperList: func(_ string) (*kms.ListAliasesInput, error) {
			return &kms.ListAliasesInput{}, nil
		},
		InputMapperSearch: func(_ context.Context, _ *kms.Client, _, query string) (*kms.ListAliasesInput, error) {
			return &kms.ListAliasesInput{
				KeyId: &query,
			}, nil
		},
		OutputMapper: aliasOutputMapper,
	}
}

var kmsAliasAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "kms-alias",
	DescriptiveName: "KMS Alias",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an alias by keyID/aliasName",
		ListDescription:   "List all aliases",
		SearchDescription: "Search aliases by keyID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_kms_alias.arn",
		},
	},
	PotentialLinks: []string{"kms-key"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
