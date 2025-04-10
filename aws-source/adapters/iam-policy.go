package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/iter"
	"go.opentelemetry.io/otel/trace"
)

type PolicyDetails struct {
	Policy       *types.Policy
	Document     *policy.Policy
	PolicyGroups []types.PolicyGroup
	PolicyRoles  []types.PolicyRole
	PolicyUsers  []types.PolicyUser
}

func policyGetFunc(ctx context.Context, client IAMClient, scope, query string) (*PolicyDetails, error) {
	// Construct the ARN from the name etc.
	a := adapterhelpers.ARN{
		ARN: arn.ARN{
			Partition: "aws",
			Service:   "iam",
			Region:    "", // IAM doesn't have a region
			AccountID: scope,
			Resource:  "policy/" + query, // The query should policyFullName which is (path + name)
		},
	}
	out, err := client.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: adapterhelpers.PtrString(a.String()),
	})
	if err != nil {
		return nil, err
	}

	details := PolicyDetails{
		Policy: out.Policy,
	}

	if out.Policy != nil {
		err := addPolicyEntities(ctx, client, &details)
		if err != nil {
			return nil, err
		}

		err = addPolicyDocument(ctx, client, &details)
		if err != nil {
			return nil, err
		}
	}

	return &details, nil
}

// Gets the current policy version and parses it, adds to the policy details
func addPolicyDocument(ctx context.Context, client IAMClient, pol *PolicyDetails) error {
	if pol.Policy == nil {
		return errors.New("policy is nil")
	}
	if pol.Policy.Arn == nil || pol.Policy.DefaultVersionId == nil {
		return errors.New("policy ARN or default version ID is nil")
	}

	out, err := client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: pol.Policy.Arn,
		VersionId: pol.Policy.DefaultVersionId,
	})
	if err != nil {
		return err
	}
	if out.PolicyVersion == nil {
		return errors.New("policy version is nil")
	}
	if out.PolicyVersion.Document == nil {
		return nil
	}

	// Save to the pointer
	pol.Document, err = ParsePolicyDocument(*out.PolicyVersion.Document)
	if err != nil {
		return fmt.Errorf("error parsing policy document: %w", err)
	}

	return nil
}

func addPolicyEntities(ctx context.Context, client IAMClient, details *PolicyDetails) error {
	var span trace.Span
	if log.GetLevel() == log.TraceLevel {
		// Only create new spans on trace level logging
		ctx, span = tracer.Start(ctx, "addPolicyEntities")
		defer span.End()
	}

	if details == nil {
		return errors.New("details is nil")
	}

	if details.Policy == nil {
		return errors.New("policy is nil")
	}

	paginator := iam.NewListEntitiesForPolicyPaginator(client, &iam.ListEntitiesForPolicyInput{
		PolicyArn: details.Policy.Arn,
	})

	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		details.PolicyGroups = append(details.PolicyGroups, out.PolicyGroups...)
		details.PolicyRoles = append(details.PolicyRoles, out.PolicyRoles...)
		details.PolicyUsers = append(details.PolicyUsers, out.PolicyUsers...)
	}

	return nil
}

func policyItemMapper(_ *string, scope string, awsItem *PolicyDetails) (*sdp.Item, error) {
	finalAttributes := struct {
		*types.Policy
		Document *policy.Policy
	}{
		Policy:   awsItem.Policy,
		Document: awsItem.Document,
	}
	attributes, err := adapterhelpers.ToAttributesWithExclude(finalAttributes)
	if err != nil {
		return nil, err
	}

	if awsItem.Policy.Path == nil || awsItem.Policy.PolicyName == nil {
		return nil, errors.New("policy Path and PolicyName must be populated")
	}

	// Combine the path and policy name to create a unique attribute
	policyFullName := *awsItem.Policy.Path + *awsItem.Policy.PolicyName

	// Trim the leading slash
	policyFullName, _ = strings.CutPrefix(policyFullName, "/")

	// Create a new attribute which is a combination of `path` and `policyName`,
	// this can then be constructed into an ARN when a user calls GET
	attributes.Set("PolicyFullName", policyFullName)

	item := sdp.Item{
		Type:            "iam-policy",
		UniqueAttribute: "PolicyFullName",
		Attributes:      attributes,
		Scope:           scope,
	}

	for _, group := range awsItem.PolicyGroups {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "iam-group",
				Query:  *group.GroupName,
				Method: sdp.QueryMethod_GET,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the group won't affect the policy
				In: false,
				// Changing the policy will affect the group
				Out: true,
			},
		})
	}

	for _, user := range awsItem.PolicyUsers {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "iam-user",
				Method: sdp.QueryMethod_GET,
				Query:  *user.UserName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the user won't affect the policy
				In: false,
				// Changing the policy will affect the user
				Out: true,
			},
		})
	}

	for _, role := range awsItem.PolicyRoles {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "iam-role",
				Method: sdp.QueryMethod_GET,
				Query:  *role.RoleName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the role won't affect the policy
				In: false,
				// Changing the policy will affect the role
				Out: true,
			},
		})
	}

	if awsItem.Document != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, LinksFromPolicy(awsItem.Document)...)
	}

	return &item, nil
}

func policyListTagsFunc(ctx context.Context, p *PolicyDetails, client IAMClient) (map[string]string, error) {
	tags := make(map[string]string)

	paginator := iam.NewListPolicyTagsPaginator(client, &iam.ListPolicyTagsInput{
		PolicyArn: p.Policy.Arn,
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

func policyListExtractor(ctx context.Context, output *iam.ListPoliciesOutput, client IAMClient) ([]*PolicyDetails, error) {
	return iter.MapErr(output.Policies, func(p *types.Policy) (*PolicyDetails, error) {
		details := PolicyDetails{
			Policy: p,
		}

		err := addPolicyEntities(ctx, client, &details)
		if err != nil {
			return &details, err
		}

		err = addPolicyDocument(ctx, client, &details)
		if err != nil {
			return &details, err
		}

		return &details, nil
	})
}

// NewPolicyAdapter Note that this policy adapter only support polices that are
// user-created due to the fact that the AWS-created ones are basically "global"
// in scope. In order to get this to work I'd have to change the way the adapter
// is implemented so that it was mart enough to handle different scopes. This
// has been added to the backlog:
// https://github.com/overmindtech/workspace/aws-adapter/issues/68
func NewIAMPolicyAdapter(client IAMClient, accountID string) *adapterhelpers.GetListAdapterV2[*iam.ListPoliciesInput, *iam.ListPoliciesOutput, *PolicyDetails, IAMClient, *iam.Options] {
	return &adapterhelpers.GetListAdapterV2[*iam.ListPoliciesInput, *iam.ListPoliciesOutput, *PolicyDetails, IAMClient, *iam.Options]{
		ItemType:               "iam-policy",
		Client:                 client,
		AccountID:              accountID,
		Region:                 "",            // IAM policies aren't tied to a region
		CacheDuration:          3 * time.Hour, // IAM has very low rate limits, we need to cache for a long time
		AdapterMetadata:        policyAdapterMetadata,
		SupportGlobalResources: true,
		InputMapperList: func(scope string) (*iam.ListPoliciesInput, error) {
			var iamScope types.PolicyScopeType
			if scope == "aws" {
				iamScope = types.PolicyScopeTypeAws
			} else {
				iamScope = types.PolicyScopeTypeLocal
			}
			return &iam.ListPoliciesInput{
				OnlyAttached: true,
				Scope:        iamScope,
			}, nil
		},
		ListFuncPaginatorBuilder: func(client IAMClient, params *iam.ListPoliciesInput) adapterhelpers.Paginator[*iam.ListPoliciesOutput, *iam.Options] {
			return iam.NewListPoliciesPaginator(client, params)
		},
		ListExtractor: policyListExtractor,
		GetFunc:       policyGetFunc,
		ItemMapper:    policyItemMapper,
		ListTagsFunc:  policyListTagsFunc,
	}
}

var policyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "iam-policy",
	DescriptiveName: "IAM Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an IAM policy by policyFullName ({path} + {policyName})",
		ListDescription:   "List all IAM policies",
		SearchDescription: "Search for IAM policies by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_iam_policy.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
		{
			TerraformQueryMap: "aws_iam_user_policy_attachment.policy_arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"iam-group", "iam-user", "iam-role"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
