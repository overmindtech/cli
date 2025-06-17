package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type IAMClient interface {
	GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
	GetPolicyVersion(ctx context.Context, params *iam.GetPolicyVersionInput, optFns ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	GetRolePolicy(ctx context.Context, params *iam.GetRolePolicyInput, optFns ...func(*iam.Options)) (*iam.GetRolePolicyOutput, error)
	GetUser(ctx context.Context, params *iam.GetUserInput, optFns ...func(*iam.Options)) (*iam.GetUserOutput, error)
	ListPolicyTags(ctx context.Context, params *iam.ListPolicyTagsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyTagsOutput, error)
	ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)

	iam.ListAttachedRolePoliciesAPIClient
	iam.ListEntitiesForPolicyAPIClient
	iam.ListGroupsForUserAPIClient
	iam.ListPoliciesAPIClient
	iam.ListRolePoliciesAPIClient
	iam.ListRolesAPIClient
	iam.ListUsersAPIClient
	iam.ListUserTagsAPIClient
}

type QueryExtractorFunc func(resource string, actions []string) []*sdp.LinkedItemQuery

// This struct extracts linked item queries from an IAM policy. It must provide
// a `RelevantResources` regex which will be checked against the resources that
// each statement is mapped to. If it matches, the `ExtractorFunc` will be
// called with the resource and actions that are allowed to be performed on that
// resource
type QueryExtractor struct {
	RelevantResources *regexp.Regexp
	ExtractorFunc     QueryExtractorFunc
}

var ssmQueryExtractor = QueryExtractor{
	RelevantResources: regexp.MustCompile("^arn:aws:ssm:"),
	ExtractorFunc: func(resource string, actions []string) []*sdp.LinkedItemQuery {
		// IAM for SSM works in a bit of a strange way: If a user has access to
		// a path, then the user can access all levels of that path. For
		// example, if a user has permission to access path /a, then the user
		// can also access /a/b. Even if a user has explicitly been denied
		// access in IAM for parameter /a/b, they can still call the
		// GetParametersByPath API operation recursively for /a and view /a/b.
		// https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-access.html
		//
		// Because of this all ARNs essential with a wildcard for the path
		a, err := adapterhelpers.ParseARN(resource)
		if err != nil {
			return nil
		}

		return []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "ssm-parameter",
					Method: sdp.QueryMethod_SEARCH,
					Query:  a.String() + "*", // Wildcard at the end
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
		}
	},
}

var fallbackQueryExtractor = QueryExtractor{
	RelevantResources: regexp.MustCompile("^arn:"),
	ExtractorFunc: func(resource string, actions []string) []*sdp.LinkedItemQuery {
		arn, err := adapterhelpers.ParseARN(resource)
		if err != nil {
			return nil
		}

		// Since this could be an ARN to anything we are going to rely
		// on the fact that we *usually* have a SEARCH method that
		// accepts ARNs
		scope := sdp.WILDCARD
		if arn.AccountID != "aws" {
			if arn.AccountID != "*" && arn.Region != "*" {
				// If we have an account and region, then use those
				scope = adapterhelpers.FormatScope(arn.AccountID, arn.Region)
			}
		}

		// We need to convert the item type from ARN format to Overmind
		// format. Since we follow a pretty strict naming convention
		// this should *usually* work. Overmind's naming conventions are
		// based on the AWS CLI, e.g. `aws ec2 describe-instances` would
		// be `ec2-instance`
		overmindType := arn.Service + "-" + arn.Type()

		// It would be good here if we had a way to definitely know what
		// type a given ARN is, but I don't think the types are 1:1 so
		// we are going to have to use a wildcard. This will cause a lot
		// of failed searches which I don't love, but it will work
		// itemType := sdp.WILDCARD

		return []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   overmindType,
					Method: sdp.QueryMethod_SEARCH,
					Query:  arn.String(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  false,
					Out: true,
				},
			},
		}
	},
}

// The ordered list of extractors to use. The first one that matches will be
// used
var extractors = []QueryExtractor{
	ssmQueryExtractor,
	fallbackQueryExtractor,
}

// Extracts linked item queries from an IAM policy. In this case we only link to
// entities that are explicitly mentioned in the policy. If we were to link to
// more you'd end up with way too many links since a policy might for example
// give read access to everything
func LinksFromPolicy(document *policy.Policy) []*sdp.LinkedItemQuery {
	// We want to link all of the resources in the policy document, as long
	// as they have a valid ARN
	queries := make([]*sdp.LinkedItemQuery, 0)

	if document == nil || document.Statements == nil {
		return queries
	}

	for _, statement := range document.Statements.Values() {
		if statement.Principal != nil {
			// If we are referencing a specific IAM user or role as the
			// principal then we should link them here
			if awsPrincipal := statement.Principal.AWS(); awsPrincipal != nil {
				for _, value := range awsPrincipal.Values() {
					// These are in the format of ARN so we'll parse them
					if arn, err := adapterhelpers.ParseARN(value); err == nil {
						var typ string
						switch arn.Type() {
						case "role":
							typ = "iam-role"
						case "user":
							typ = "iam-user"
						}

						if typ != "" {
							queries = append(queries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   typ,
									Method: sdp.QueryMethod_SEARCH,
									Query:  arn.String(),
									Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
								},
								BlastPropagation: &sdp.BlastPropagation{
									// If a user or role iex explicitly
									// referenced, I think it's reasonable to
									// assume that they are tightly bound
									In:  true,
									Out: true,
								},
							})
						}
					}
				}
			}
		}

		if statement.Resource != nil {
			for _, resource := range statement.Resource.Values() {
				// Try to extract links from the references resource using the
				// configurable extractors
				for _, extractor := range extractors {
					if extractor.RelevantResources != nil && extractor.RelevantResources.MatchString(resource) {
						if statement.Action == nil || len(statement.Action.Values()) == 0 {
							// If there is no action, then we can't extract
							// anything from this resource
							continue
						}
						queries = append(queries, extractor.ExtractorFunc(resource, statement.Action.Values())...)

						// Only use the first one that matches
						break
					}
				}
			}
		}
	}

	return queries
}

// Parses an IAM policy in it's URL-encoded embedded form
func ParsePolicyDocument(encoded string) (*policy.Policy, error) {
	// Decode the policy document which is RFC 3986 URL encoded
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode policy document: %w", err)
	}

	// Unmarshal the JSON
	policyDocument := policy.Policy{}
	err = json.Unmarshal([]byte(decoded), &policyDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy document: %w", err)
	}

	return &policyDocument, nil
}
