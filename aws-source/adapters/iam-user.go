package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type UserDetails struct {
	User       *types.User
	UserGroups []types.Group
}

func userGetFunc(ctx context.Context, client IAMClient, _, query string) (*UserDetails, error) {
	out, err := client.GetUser(ctx, &iam.GetUserInput{
		UserName: &query,
	})

	if err != nil {
		return nil, err
	}

	details := UserDetails{
		User: out.User,
	}

	if out.User != nil {
		err = enrichUser(ctx, client, &details)
		if err != nil {
			return nil, fmt.Errorf("failed to enrich user %w", err)
		}
	}

	return &details, nil
}

// enrichUser Enriches the user with group and tag info
func enrichUser(ctx context.Context, client IAMClient, userDetails *UserDetails) error {
	var err error

	userDetails.UserGroups, err = getUserGroups(ctx, client, userDetails.User.UserName)

	if err != nil {
		return err
	}

	return nil
}

// Gets all of the groups that a user is in
func getUserGroups(ctx context.Context, client IAMClient, userName *string) ([]types.Group, error) {
	var out *iam.ListGroupsForUserOutput
	var err error
	groups := make([]types.Group, 0)

	paginator := iam.NewListGroupsForUserPaginator(client, &iam.ListGroupsForUserInput{
		UserName: userName,
	})

	for paginator.HasMorePages() {
		out, err = paginator.NextPage(ctx)

		if err != nil {
			return nil, err

		}

		groups = append(groups, out.Groups...)
	}

	return groups, nil
}

func userItemMapper(_ *string, scope string, awsItem *UserDetails) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem.User)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "iam-user",
		UniqueAttribute: "UserName",
		Attributes:      attributes,
		Scope:           scope,
	}

	for _, group := range awsItem.UserGroups {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "iam-group",
				Method: sdp.QueryMethod_GET,
				Query:  *group.GroupName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the group can affect the user
				In: true,
				// Changing the user won't affect the group
				Out: false,
			},
		})
	}

	return &item, nil
}

func userListTagsFunc(ctx context.Context, u *UserDetails, client IAMClient) (map[string]string, error) {
	tags := make(map[string]string)

	paginator := iam.NewListUserTagsPaginator(client, &iam.ListUserTagsInput{
		UserName: u.User.UserName,
	})

	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)

		if err != nil {
			return adapterhelpers.HandleTagsError(ctx, err), nil
		}

		for _, tag := range out.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	return tags, nil
}

func NewIAMUserAdapter(client IAMClient, accountID string) *adapterhelpers.GetListAdapterV2[*iam.ListUsersInput, *iam.ListUsersOutput, *UserDetails, IAMClient, *iam.Options] {
	return &adapterhelpers.GetListAdapterV2[*iam.ListUsersInput, *iam.ListUsersOutput, *UserDetails, IAMClient, *iam.Options]{
		ItemType:      "iam-user",
		Client:        client,
		CacheDuration: 3 * time.Hour, // IAM has very low rate limits, we need to cache for a long time
		AccountID:     accountID,
		GetFunc: func(ctx context.Context, client IAMClient, scope, query string) (*UserDetails, error) {
			return userGetFunc(ctx, client, scope, query)
		},
		InputMapperList: func(scope string) (*iam.ListUsersInput, error) {
			return &iam.ListUsersInput{}, nil
		},
		ListFuncPaginatorBuilder: func(client IAMClient, input *iam.ListUsersInput) adapterhelpers.Paginator[*iam.ListUsersOutput, *iam.Options] {
			return iam.NewListUsersPaginator(client, input)
		},
		ListExtractor: func(ctx context.Context, output *iam.ListUsersOutput, client IAMClient) ([]*UserDetails, error) {
			userDetails := make([]*UserDetails, 0, len(output.Users))

			for i := range output.Users {
				details := UserDetails{
					User: &output.Users[i],
				}

				err := enrichUser(ctx, client, &details)
				if err != nil {
					return nil, fmt.Errorf("failed to enrich user %s: %w", *details.User.UserName, err)
				}

				userDetails = append(userDetails, &details)
			}

			return userDetails, nil
		},
		ItemMapper:      userItemMapper,
		ListTagsFunc:    userListTagsFunc,
		AdapterMetadata: iamUserAdapterMetadata,
	}
}

var iamUserAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "iam-user",
	DescriptiveName: "IAM User",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an IAM user by name",
		ListDescription:   "List all IAM users",
		SearchDescription: "Search for IAM users by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_iam_user.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
		{
			TerraformQueryMap: "aws_iam_user_group_membership.user",
			TerraformMethod:   sdp.QueryMethod_GET,
		},
	},
	PotentialLinks: []string{"iam-group"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
