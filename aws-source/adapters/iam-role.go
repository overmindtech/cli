package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/sourcegraph/conc/iter"
)

type RoleDetails struct {
	Role             *types.Role
	EmbeddedPolicies []embeddedPolicy
	AttachedPolicies []types.AttachedPolicy
}

func roleGetFunc(ctx context.Context, client IAMClient, _, query string) (*RoleDetails, error) {
	out, err := client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: &query,
	})

	if err != nil {
		return nil, err
	}

	details := RoleDetails{
		Role: out.Role,
	}

	err = enrichRole(ctx, client, &details)

	if err != nil {
		return nil, err
	}

	return &details, nil
}

func enrichRole(ctx context.Context, client IAMClient, roleDetails *RoleDetails) error {
	var err error

	// In this section we want to get the embedded polices, and determine links
	// to the attached policies

	// Get embedded policies
	roleDetails.EmbeddedPolicies, err = getEmbeddedPolicies(ctx, client, *roleDetails.Role.RoleName)

	if err != nil {
		return err
	}

	// Get the attached policies and create links to these
	roleDetails.AttachedPolicies, err = getAttachedPolicies(ctx, client, *roleDetails.Role.RoleName)

	if err != nil {
		return err
	}

	return nil
}

type embeddedPolicy struct {
	Name     string
	Document *policy.Policy
}

// getEmbeddedPolicies returns a list of inline policies embedded in the role
func getEmbeddedPolicies(ctx context.Context, client IAMClient, roleName string) ([]embeddedPolicy, error) {
	policiesPaginator := iam.NewListRolePoliciesPaginator(client, &iam.ListRolePoliciesInput{
		RoleName: &roleName,
	})
	ctx, span := tracer.Start(ctx, "getEmbeddedPolicies")
	defer span.End()

	policies := make([]embeddedPolicy, 0)

	for policiesPaginator.HasMorePages() {
		out, err := policiesPaginator.NextPage(ctx)

		if err != nil {
			return nil, err
		}

		for _, policyName := range out.PolicyNames {
			embeddedPolicy, err := getRolePolicyDetails(ctx, client, roleName, policyName)

			if err != nil {
				// Ignore these errors
				continue
			}

			policies = append(policies, *embeddedPolicy)
		}
	}

	return policies, nil
}

func getRolePolicyDetails(ctx context.Context, client IAMClient, roleName string, policyName string) (*embeddedPolicy, error) {
	ctx, span := tracer.Start(ctx, "getRolePolicyDetails")
	defer span.End()
	policy, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
		RoleName:   &roleName,
		PolicyName: &policyName,
	})

	if err != nil {
		return nil, err
	}

	if policy == nil || policy.PolicyDocument == nil {
		return nil, errors.New("policy document not found")
	}

	policyDoc, err := ParsePolicyDocument(*policy.PolicyDocument)
	if err != nil {
		return nil, fmt.Errorf("error parsing policy document: %w", err)
	}

	return &embeddedPolicy{
		Name:     policyName,
		Document: policyDoc,
	}, nil
}

// getAttachedPolicies Gets the attached policies for a role, these are actual
// managed policies that can be linked to rather than embedded ones
func getAttachedPolicies(ctx context.Context, client IAMClient, roleName string) ([]types.AttachedPolicy, error) {
	paginator := iam.NewListAttachedRolePoliciesPaginator(client, &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	})

	attachedPolicies := make([]types.AttachedPolicy, 0)

	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)

		if err != nil {
			return nil, err
		}

		attachedPolicies = append(attachedPolicies, out.AttachedPolicies...)
	}

	return attachedPolicies, nil
}

func roleItemMapper(_ *string, scope string, awsItem *RoleDetails) (*sdp.Item, error) {
	enrichedRole := struct {
		*types.Role
		EmbeddedPolicies []embeddedPolicy
		// This is a replacement for the URL-encoded policy document so that the
		// user can see the policy
		AssumeRolePolicyDocument *policy.Policy
	}{
		Role:             awsItem.Role,
		EmbeddedPolicies: awsItem.EmbeddedPolicies,
	}

	// Parse the encoded policy document
	if awsItem.Role.AssumeRolePolicyDocument != nil {
		policyDoc, err := ParsePolicyDocument(*awsItem.Role.AssumeRolePolicyDocument)
		if err != nil {
			return nil, err
		}

		enrichedRole.AssumeRolePolicyDocument = policyDoc
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(enrichedRole)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "iam-role",
		UniqueAttribute: "RoleName",
		Attributes:      attributes,
		Scope:           scope,
	}

	for _, policy := range awsItem.AttachedPolicies {
		if policy.PolicyArn != nil {
			if a, err := adapterhelpers.ParseARN(*policy.PolicyArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-policy",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *policy.PolicyArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the policy will affect the role
						In: true,
						// Changing the role won't affect the policy
						Out: false,
					},
				})
			}
		}
	}

	// Extract links from policy documents
	for _, policy := range awsItem.EmbeddedPolicies {
		item.LinkedItemQueries = append(item.LinkedItemQueries, LinksFromPolicy(policy.Document)...)
	}

	// Extract links from the assume role policy document
	if enrichedRole.AssumeRolePolicyDocument != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, LinksFromPolicy(enrichedRole.AssumeRolePolicyDocument)...)
	}

	return &item, nil
}

func roleListTagsFunc(ctx context.Context, r *RoleDetails, client IAMClient) (map[string]string, error) {
	tags := make(map[string]string)

	paginator := iam.NewListRoleTagsPaginator(client, &iam.ListRoleTagsInput{
		RoleName: r.Role.RoleName,
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

func NewIAMRoleAdapter(client IAMClient, accountID string) *adapterhelpers.GetListAdapterV2[*iam.ListRolesInput, *iam.ListRolesOutput, *RoleDetails, IAMClient, *iam.Options] {
	return &adapterhelpers.GetListAdapterV2[*iam.ListRolesInput, *iam.ListRolesOutput, *RoleDetails, IAMClient, *iam.Options]{
		ItemType:      "iam-role",
		Client:        client,
		CacheDuration: 3 * time.Hour, // IAM has very low rate limits, we need to cache for a long time
		AccountID:     accountID,
		GetFunc: func(ctx context.Context, client IAMClient, scope, query string) (*RoleDetails, error) {
			return roleGetFunc(ctx, client, scope, query)
		},
		InputMapperList: func(scope string) (*iam.ListRolesInput, error) {
			return &iam.ListRolesInput{}, nil
		},
		ListFuncPaginatorBuilder: func(client IAMClient, input *iam.ListRolesInput) adapterhelpers.Paginator[*iam.ListRolesOutput, *iam.Options] {
			return iam.NewListRolesPaginator(client, input)
		},
		ListExtractor: func(ctx context.Context, output *iam.ListRolesOutput, client IAMClient) ([]*RoleDetails, error) {
			roles := make([]*RoleDetails, 0)
			mapper := iter.Mapper[types.Role, *RoleDetails]{
				MaxGoroutines: 100,
			}

			newRoles, err := mapper.MapErr(output.Roles, func(role *types.Role) (*RoleDetails, error) {
				details := RoleDetails{
					Role: role,
				}

				err := enrichRole(ctx, client, &details)
				if err != nil {
					return nil, err
				}

				return &details, nil
			})

			if err != nil {
				return nil, err
			}

			roles = append(roles, newRoles...)
			return roles, nil
		},
		ItemMapper:      roleItemMapper,
		ListTagsFunc:    roleListTagsFunc,
		AdapterMetadata: roleAdapterMetadata,
	}
}

var roleAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "iam-role",
	DescriptiveName: "IAM Role",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an IAM role by name",
		ListDescription:   "List all IAM roles",
		SearchDescription: "Search for IAM roles by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_iam_role.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"iam-policy"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
