package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func iamInstanceProfileAssociationOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeIamInstanceProfileAssociationsInput, output *ec2.DescribeIamInstanceProfileAssociationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, assoc := range output.IamInstanceProfileAssociations {
		attributes, err := adapterhelpers.ToAttributesWithExclude(assoc)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ec2-iam-instance-profile-association",
			UniqueAttribute: "AssociationId",
			Attributes:      attributes,
			Scope:           scope,
		}

		if assoc.IamInstanceProfile != nil && assoc.IamInstanceProfile.Arn != nil {
			if arn, err := adapterhelpers.ParseARN(*assoc.IamInstanceProfile.Arn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-instance-profile",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *assoc.IamInstanceProfile.Arn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the profile will affect this
						In: true,
						// We can't affect the profile
						Out: false,
					},
				})
			}
		}

		if assoc.InstanceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-instance",
					Method: sdp.QueryMethod_GET,
					Query:  *assoc.InstanceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the instance will not affect the association
					In: false,
					// changes to the association will affect the instance
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

// NewIamInstanceProfileAssociationAdapter Creates a new adapter for aws-IamInstanceProfileAssociation resources
func NewEC2IamInstanceProfileAssociationAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeIamInstanceProfileAssociationsInput, *ec2.DescribeIamInstanceProfileAssociationsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeIamInstanceProfileAssociationsInput, *ec2.DescribeIamInstanceProfileAssociationsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-iam-instance-profile-association",
		AdapterMetadata: iamInstanceProfileAssociationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeIamInstanceProfileAssociationsInput) (*ec2.DescribeIamInstanceProfileAssociationsOutput, error) {
			return client.DescribeIamInstanceProfileAssociations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*ec2.DescribeIamInstanceProfileAssociationsInput, error) {
			return &ec2.DescribeIamInstanceProfileAssociationsInput{
				AssociationIds: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*ec2.DescribeIamInstanceProfileAssociationsInput, error) {
			return &ec2.DescribeIamInstanceProfileAssociationsInput{}, nil
		},
		OutputMapper: iamInstanceProfileAssociationOutputMapper,
	}
}

var iamInstanceProfileAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-iam-instance-profile-association",
	DescriptiveName: "IAM Instance Profile Association",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an IAM Instance Profile Association by ID",
		ListDescription:   "List all IAM Instance Profile Associations",
		SearchDescription: "Search IAM Instance Profile Associations by ARN",
	},
	PotentialLinks: []string{"iam-instance-profile", "ec2-instance"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
