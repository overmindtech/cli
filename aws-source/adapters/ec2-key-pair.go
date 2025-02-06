package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func keyPairInputMapperGet(scope string, query string) (*ec2.DescribeKeyPairsInput, error) {
	return &ec2.DescribeKeyPairsInput{
		KeyNames: []string{
			query,
		},
	}, nil
}

func keyPairInputMapperList(scope string) (*ec2.DescribeKeyPairsInput, error) {
	return &ec2.DescribeKeyPairsInput{}, nil
}

func keyPairOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeKeyPairsInput, output *ec2.DescribeKeyPairsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, gw := range output.KeyPairs {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(gw, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-key-pair",
			UniqueAttribute: "KeyName",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(gw.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2KeyPairAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeKeyPairsInput, *ec2.DescribeKeyPairsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeKeyPairsInput, *ec2.DescribeKeyPairsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-key-pair",
		AdapterMetadata: keyPairAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return client.DescribeKeyPairs(ctx, input)
		},
		InputMapperGet:  keyPairInputMapperGet,
		InputMapperList: keyPairInputMapperList,
		OutputMapper:    keyPairOutputMapper,
	}
}

var keyPairAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-key-pair",
	DescriptiveName: "Key Pair",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a key pair by name",
		ListDescription:   "List all key pairs",
		SearchDescription: "Search for key pairs by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_key_pair.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
