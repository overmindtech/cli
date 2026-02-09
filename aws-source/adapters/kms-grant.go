package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/kms"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"

	log "github.com/sirupsen/logrus"
)

func grantOutputMapper(ctx context.Context, _ *kms.Client, scope string, _ *kms.ListGrantsInput, output *kms.ListGrantsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, grant := range output.Grants {
		attributes, err := ToAttributesWithExclude(grant, "tags")
		if err != nil {
			return nil, err
		}

		// This should never happen.
		if grant.GrantId == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "grantId is nil",
			}
		}

		arn, errA := ParseARN(*grant.KeyId)
		if errA != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf("failed to parse keyID: %s", *grant.KeyId),
			}
		}

		keyID := arn.ResourceID()

		// The uniqueAttributeValue for this is the combination of the keyID and grantId
		// i.e., "cf68415c-f4ae-48f2-87a7-3b52ce/grant-id"
		err = attributes.Set("UniqueName", fmt.Sprintf("%s/%s", keyID, *grant.GrantId))
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "kms-grant",
			UniqueAttribute: "UniqueName",
			Attributes:      attributes,
			Scope:           scope,
		}

		scope = FormatScope(arn.AccountID, arn.Region)

		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "kms-key",
				Method: sdp.QueryMethod_GET,
				Query:  keyID,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				// Adding or revoking/retiring a grant can allow or deny permission to the KMS key for the grantee.
				In:  true,
				Out: true,
			},
		})

		var principals []string
		if grant.GranteePrincipal != nil {
			principals = append(principals, *grant.GranteePrincipal)
		}

		if grant.RetiringPrincipal != nil {
			principals = append(principals, *grant.RetiringPrincipal)
		}

		// Valid principals include
		// - Amazon Web Services accounts
		// - IAM users,
		// - IAM roles,
		// - federated users,
		// - assumed role users.
		// principals: https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-arns
		/*
			arn:aws:iam::account:root
			arn:aws:iam::account:user/user-name-with-path
			arn:aws:iam::account:role/role-name-with-path
			arn:aws:sts::account:federated-user/user-name
			arn:aws:sts::account:assumed-role/role-name/role-session-name
			arn:aws:sts::account:self

			dynamodb.us-west-2.amazonaws.com

			The following are not supported (we skip them silently):
				- arn:aws:iam::account:root
				- arn:aws:sts::account:federated-user/user-name
				- arn:aws:sts::account:assumed-role/role-name/role-session-name
				- arn:aws:sts::account:self
				- Service principals like dynamodb.us-west-2.amazonaws.com (not ARNs, not linkable)
		*/

		for _, principal := range principals {
			// Skip AWS service principals (e.g. "rds.eu-west-2.amazonaws.com",
			// "dynamodb.us-west-2.amazonaws.com"). These are DNS-style identifiers
			// for AWS services, not ARNs, and are not linkable to other items.
			if isAWSServicePrincipal(principal) {
				log.WithFields(log.Fields{
					"input": principal,
					"scope": scope,
				}).Debug("Skipping AWS service principal (not linkable)")

				continue
			}

			lIQ := &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Method: sdp.QueryMethod_GET,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					// Adding or revoking/retiring a grant can allow or deny permission to the KMS key for the grantee.
					// Or, disabling a role will make the grant redundant.
					In:  true,
					Out: true,
				},
			}

			arn, errA := ParseARN(principal)
			if errA != nil {
				log.WithFields(log.Fields{
					"error": errA,
					"input": principal,
					"scope": scope,
				}).Warn("Error parsing principal ARN")

				continue
			}

			switch arn.Service {
			case "iam":
				adapter, query := iamSourceAndQuery(arn.Resource)
				switch adapter {
				case "user":
					lIQ.Query.Type = "iam-user"
					lIQ.Query.Query = query
				case "role":
					lIQ.Query.Type = "iam-role"
					lIQ.Query.Query = query
				default:
					log.WithFields(log.Fields{
						"input": principal,
						"scope": scope,
					}).Warn("Error unsupported iam adapter")

					continue
				}
			default:
				log.WithFields(log.Fields{
					"input": principal,
					"scope": scope,
				}).Warn("Error ARN service not supported")

				continue
			}

			item.LinkedItemQueries = append(item.LinkedItemQueries, lIQ)
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewKMSGrantAdapter(client *kms.Client, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*kms.ListGrantsInput, *kms.ListGrantsOutput, *kms.Client, *kms.Options] {
	return &DescribeOnlyAdapter[*kms.ListGrantsInput, *kms.ListGrantsOutput, *kms.Client, *kms.Options]{
		ItemType:        "kms-grant",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: grantAdapterMetadata,
		cache:        cache,
		DescribeFunc: func(ctx context.Context, client *kms.Client, input *kms.ListGrantsInput) (*kms.ListGrantsOutput, error) {
			return client.ListGrants(ctx, input)
		},
		InputMapperGet: func(_, query string) (*kms.ListGrantsInput, error) {
			// query must be in the format of: the keyID/grantId
			// i.e., "cf68415c-f4ae-48f2-87a7-3b52ce/grant-id"
			tmp := strings.Split(query, "/") // [keyID, grantId]
			if len(tmp) < 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("query must be in the format of: the keyID/grantId, but found: %s", query),
				}
			}

			return &kms.ListGrantsInput{
				KeyId:   &tmp[0],                                              // keyID
				GrantId: PtrString(strings.Join(tmp[1:], "/")), // grantId
			}, nil
		},
		UseListForGet: true,
		InputMapperList: func(_ string) (*kms.ListGrantsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for kms-grant, use search",
			}
		},
		InputMapperSearch: func(_ context.Context, _ *kms.Client, _, query string) (*kms.ListGrantsInput, error) {
			return &kms.ListGrantsInput{
				KeyId: &query,
			}, nil
		},
		OutputMapper: grantOutputMapper,
	}
}

var grantAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "kms-grant",
	DescriptiveName: "KMS Grant",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a grant by keyID/grantId",
		SearchDescription: "Search grants by keyID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_kms_grant.grant_id",
		},
	},
	PotentialLinks: []string{"kms-key", "iam-user", "iam-role"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})

// example: user/user-name-with-path
func iamSourceAndQuery(resource string) (string, string) {
	tmp := strings.Split(resource, "/") // [user, user-name-with-path]

	adapter := tmp[0]
	query := strings.Join(tmp[1:], "/")

	return adapter, query // user, user-name-with-path
}

// isAWSServicePrincipal returns true if the principal is an AWS service
// principal (e.g. "rds.eu-west-2.amazonaws.com", "dynamodb.us-west-2.amazonaws.com").
// These are DNS-style identifiers used by AWS services to assume roles or access
// resources, and are not ARNs.
func isAWSServicePrincipal(principal string) bool {
	// Service principals don't start with "arn:" and end with a partition-specific
	// DNS suffix.
	if strings.HasPrefix(principal, "arn:") {
		return false
	}

	// Check all AWS partition DNS suffixes using the shared list
	for _, suffix := range GetAllAWSPartitionDNSSuffixes() {
		if strings.HasSuffix(principal, "."+suffix) {
			return true
		}
	}

	return false
}
