package adapters

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func instanceProfileGetFunc(ctx context.Context, client *iam.Client, _, query string) (*types.InstanceProfile, error) {
	out, err := client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: &query,
	})

	if err != nil {
		return nil, err
	}

	return out.InstanceProfile, nil
}

func instanceProfileItemMapper(_ *string, scope string, awsItem *types.InstanceProfile) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "iam-instance-profile",
		UniqueAttribute: "InstanceProfileName",
		Attributes:      attributes,
		Scope:           scope,
	}

	for _, role := range awsItem.Roles {
		if arn, err := adapterhelpers.ParseARN(*role.Arn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *role.Arn,
					Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the role will affect this
					In: true,
					// We can't affect the role
					Out: false,
				},
			})
		}

		if role.PermissionsBoundary != nil {
			if arn, err := adapterhelpers.ParseARN(*role.PermissionsBoundary.PermissionsBoundaryArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-policy",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *role.PermissionsBoundary.PermissionsBoundaryArn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the policy will affect this
						In: true,
						// We can't affect the policy
						Out: false,
					},
				})
			}
		}
	}

	return &item, nil
}

func instanceProfileListTagsFunc(ctx context.Context, ip *types.InstanceProfile, client *iam.Client) map[string]string {
	tags := make(map[string]string)

	paginator := iam.NewListInstanceProfileTagsPaginator(client, &iam.ListInstanceProfileTagsInput{
		InstanceProfileName: ip.InstanceProfileName,
	})

	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)

		if err != nil {
			return adapterhelpers.HandleTagsError(ctx, err)
		}

		for _, tag := range out.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	return tags
}

func NewIAMInstanceProfileAdapter(client *iam.Client, accountID string) *adapterhelpers.GetListAdapterV2[*iam.ListInstanceProfilesInput, *iam.ListInstanceProfilesOutput, *types.InstanceProfile, *iam.Client, *iam.Options] {
	return &adapterhelpers.GetListAdapterV2[*iam.ListInstanceProfilesInput, *iam.ListInstanceProfilesOutput, *types.InstanceProfile, *iam.Client, *iam.Options]{
		ItemType:        "iam-instance-profile",
		Client:          client,
		CacheDuration:   3 * time.Hour, // IAM has very low rate limits, we need to cache for a long time
		AccountID:       accountID,
		AdapterMetadata: instanceProfileAdapterMetadata,
		GetFunc: func(ctx context.Context, client *iam.Client, scope, query string) (*types.InstanceProfile, error) {
			return instanceProfileGetFunc(ctx, client, scope, query)
		},
		InputMapperList: func(scope string) (*iam.ListInstanceProfilesInput, error) {
			return &iam.ListInstanceProfilesInput{}, nil
		},
		ListFuncPaginatorBuilder: func(client *iam.Client, params *iam.ListInstanceProfilesInput) adapterhelpers.Paginator[*iam.ListInstanceProfilesOutput, *iam.Options] {
			return iam.NewListInstanceProfilesPaginator(client, params)
		},
		ListExtractor: func(_ context.Context, output *iam.ListInstanceProfilesOutput, _ *iam.Client) ([]*types.InstanceProfile, error) {
			profiles := make([]*types.InstanceProfile, 0, len(output.InstanceProfiles))
			for i := range output.InstanceProfiles {
				profiles = append(profiles, &output.InstanceProfiles[i])
			}
			return profiles, nil
		},
		ListTagsFunc: func(ctx context.Context, ip *types.InstanceProfile, c *iam.Client) (map[string]string, error) {
			return instanceProfileListTagsFunc(ctx, ip, c), nil
		},
		ItemMapper: instanceProfileItemMapper,
	}
}

var instanceProfileAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "iam-instance-profile",
	DescriptiveName: "IAM Instance Profile",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an IAM instance profile by name",
		ListDescription:   "List all IAM instance profiles",
		SearchDescription: "Search IAM instance profiles by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_iam_instance_profile.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"iam-role", "iam-policy"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
