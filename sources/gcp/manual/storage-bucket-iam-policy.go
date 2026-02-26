package manual

import (
	"context"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Storage Bucket IAM Policy adapter: one item per bucket representing the bucket's full IAM policy.
// Uses the Storage Bucket getIamPolicy V3 API. All Terraform bucket IAM resources (binding, member, policy) map to this item.
// See: https://cloud.google.com/storage/docs/json_api/v1/buckets/getIamPolicy

var (
	StorageBucketIAMPolicyLookupByBucket = shared.NewItemTypeLookup("bucket", gcpshared.StorageBucketIAMPolicy)
)

type storageBucketIAMPolicyWrapper struct {
	client gcpshared.StorageBucketIAMPolicyGetter
	*gcpshared.ProjectBase
}

// NewStorageBucketIAMPolicy creates a SearchableWrapper for Storage Bucket IAM policy (one item per bucket).
func NewStorageBucketIAMPolicy(client gcpshared.StorageBucketIAMPolicyGetter, locations []gcpshared.LocationInfo) sources.SearchableWrapper {
	return &storageBucketIAMPolicyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.StorageBucketIAMPolicy,
		),
	}
}

func (w *storageBucketIAMPolicyWrapper) IAMPermissions() []string {
	return []string{"storage.buckets.getIamPolicy"}
}

func (w *storageBucketIAMPolicyWrapper) PredefinedRole() string {
	return "overmind_custom_role"
}

func (w *storageBucketIAMPolicyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.StorageBucket,
		gcpshared.IAMServiceAccount,
		gcpshared.IAMRole,
		gcpshared.ComputeProject,
		stdlib.NetworkDNS,
	)
}

func (w *storageBucketIAMPolicyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_storage_bucket_iam_binding.bucket",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_storage_bucket_iam_member.bucket",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_storage_bucket_iam_policy.bucket",
		},
	}
}

func (w *storageBucketIAMPolicyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageBucketIAMPolicyLookupByBucket,
	}
}

func (w *storageBucketIAMPolicyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{StorageBucketIAMPolicyLookupByBucket},
	}
}

func (w *storageBucketIAMPolicyWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := w.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}
	if len(queryParts) < 1 || queryParts[0] == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "GET requires bucket name",
		}
	}
	bucketName := queryParts[0]

	bindings, getErr := w.client.GetBucketIAMPolicy(ctx, bucketName)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, w.Type())
	}

	return w.policyToItem(location, bucketName, bindings)
}

func (w *storageBucketIAMPolicyWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := w.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}
	if len(queryParts) < 1 || queryParts[0] == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "SEARCH requires bucket name",
		}
	}
	bucketName := queryParts[0]

	bindings, getErr := w.client.GetBucketIAMPolicy(ctx, bucketName)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, w.Type())
	}

	item, qErr := w.policyToItem(location, bucketName, bindings)
	if qErr != nil {
		return nil, qErr
	}
	return []*sdp.Item{item}, nil
}

// policyBinding is the serialized shape of one binding in the policy item attributes.
type policyBinding struct {
	Role                 string   `json:"role"`
	Members              []string `json:"members"`
	ConditionExpression  string   `json:"conditionExpression,omitempty"`
	ConditionTitle       string   `json:"conditionTitle,omitempty"`
	ConditionDescription string   `json:"conditionDescription,omitempty"`
}

// policyToItem builds one SDP item for the bucket's IAM policy and adds linked item queries from all bindings.
func (w *storageBucketIAMPolicyWrapper) policyToItem(location gcpshared.LocationInfo, bucketName string, bindings []gcpshared.BucketIAMBinding) (*sdp.Item, *sdp.QueryError) {
	policyBindings := make([]policyBinding, 0, len(bindings))
	for _, b := range bindings {
		policyBindings = append(policyBindings, policyBinding{
			Role:                 b.Role,
			Members:              b.Members,
			ConditionExpression:  b.ConditionExpression,
			ConditionTitle:       b.ConditionTitle,
			ConditionDescription: b.ConditionDescription,
		})
	}

	type policyAttrs struct {
		Bucket   string          `json:"bucket"`
		Bindings []policyBinding `json:"bindings"`
	}
	attrs, err := shared.ToAttributesWithExclude(policyAttrs{Bucket: bucketName, Bindings: policyBindings})
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), w.Type())
	}
	if err = attrs.Set("uniqueAttr", bucketName); err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), w.Type())
	}

	item := &sdp.Item{
		Type:            gcpshared.StorageBucketIAMPolicy.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attrs,
		Scope:           location.ToScope(),
	}

	// Link to StorageBucket (In: true, Out: true)
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.StorageBucket.String(),
			Method: sdp.QueryMethod_GET,
			Query:  bucketName,
			Scope:  location.ProjectID,
		},
	})

	// Collect unique linked SAs, projects, domains, and custom IAM roles across all bindings.
	linkedSAs := make(map[string]string) // email -> projectID
	linkedProjects := make(map[string]struct{})
	linkedDomains := make(map[string]struct{})
	linkedRoles := make(map[string]map[string]struct{}) // projectID -> set of roleIDs

	for _, b := range bindings {
		// Custom roles are in the form projects/{project}/roles/{roleId}; predefined roles are roles/...
		if projectID, roleID := extractCustomRoleProjectAndID(b.Role); projectID != "" && roleID != "" {
			if linkedRoles[projectID] == nil {
				linkedRoles[projectID] = make(map[string]struct{})
			}
			linkedRoles[projectID][roleID] = struct{}{}
		}
		for _, member := range b.Members {
			saEmail := extractServiceAccountEmailFromMember(member)
			if saEmail != "" {
				projectID := extractProjectFromServiceAccountEmail(saEmail)
				if projectID != "" && !isGoogleManagedServiceAccountDomain(projectID) {
					linkedSAs[saEmail] = projectID
				}
			}
			projectID := extractProjectIDFromProjectPrincipalMember(member)
			if projectID != "" {
				linkedProjects[projectID] = struct{}{}
			}
			domainName := extractDomainFromDomainMember(member)
			if domainName != "" {
				linkedDomains[domainName] = struct{}{}
			}
		}
	}

	for saEmail, projectID := range linkedSAs {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.IAMServiceAccount.String(),
				Method: sdp.QueryMethod_GET,
				Query:  saEmail,
				Scope:  projectID,
			},
		})
	}
	for projectID, roleIDs := range linkedRoles {
		for roleID := range roleIDs {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.IAMRole.String(),
					Method: sdp.QueryMethod_GET,
					Query:  roleID,
					Scope:  projectID,
				},
			})
		}
	}
	for projectID := range linkedProjects {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeProject.String(),
				Method: sdp.QueryMethod_GET,
				Query:  projectID,
				Scope:  projectID,
			},
		})
	}
	for domainName := range linkedDomains {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  domainName,
				Scope:  "global",
			},
		})
	}

	return item, nil
}

// extractCustomRoleProjectAndID parses a custom IAM role reference "projects/{project}/roles/{roleId}"
// and returns (projectID, roleID). For predefined roles (e.g. "roles/storage.objectViewer") returns ("", "").
func extractCustomRoleProjectAndID(role string) (projectID, roleID string) {
	const prefix = "projects/"
	const suffix = "/roles/"
	if !strings.HasPrefix(role, prefix) || !strings.Contains(role, suffix) {
		return "", ""
	}
	rest := strings.TrimPrefix(role, prefix)
	before, after, ok := strings.Cut(rest, suffix)
	if !ok {
		return "", ""
	}
	projectID = before
	roleID = after
	if projectID == "" || roleID == "" {
		return "", ""
	}
	return projectID, roleID
}

// extractDomainFromDomainMember returns the domain for "domain:example.com" or
// "deleted:domain:example.com", or "" otherwise. The value is a DNS name.
// For deleted members, any "?uid=..." suffix is stripped so the result is a valid DNS link.
func extractDomainFromDomainMember(member string) string {
	var domain string
	if after, ok := strings.CutPrefix(member, "deleted:domain:"); ok {
		domain = after
	} else if after, ok := strings.CutPrefix(member, "domain:"); ok {
		domain = after
	} else {
		return ""
	}
	// Deleted domain members can include "?uid=123456789"; strip so link uses the actual domain.
	if idx := strings.Index(domain, "?"); idx != -1 {
		domain = domain[:idx]
	}
	return domain
}

// extractProjectIDFromProjectPrincipalMember returns the project ID for project principal members
// (projectOwner:projectId, projectEditor:projectId, projectViewer:projectId), or "" otherwise.
func extractProjectIDFromProjectPrincipalMember(member string) string {
	for _, prefix := range []string{"projectOwner:", "projectEditor:", "projectViewer:"} {
		if after, ok := strings.CutPrefix(member, prefix); ok {
			return after
		}
	}
	return ""
}

// extractServiceAccountEmailFromMember returns the email for "serviceAccount:email" or "deleted:serviceAccount:email", or "" if not a service account member.
// For deleted members, any "?uid=..." suffix is stripped so the result is a valid IAMServiceAccount lookup query (email only).
func extractServiceAccountEmailFromMember(member string) string {
	var email string
	if after, ok := strings.CutPrefix(member, "deleted:serviceAccount:"); ok {
		email = after
	} else if after, ok := strings.CutPrefix(member, "serviceAccount:"); ok {
		email = after
	} else {
		return ""
	}
	// Deleted SAs can include "?uid=123456789"; strip query part so link uses the actual SA email.
	if idx := strings.Index(email, "?"); idx != -1 {
		email = email[:idx]
	}
	return email
}

// extractProjectFromServiceAccountEmail extracts project ID from "name@project.iam.gserviceaccount.com".
// Only project-scoped SAs use that domain; developer.gserviceaccount.com and appspot.gserviceaccount.com
// use a shared domain where the first label is not a project ID, so we return "" to avoid invalid links.
// For Google-managed SAs (e.g. name@gcp-sa-logging.iam.gserviceaccount.com) use isGoogleManagedServiceAccountDomain to skip.
func extractProjectFromServiceAccountEmail(email string) string {
	_, after, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	domain := after
	// Only use first label as project when domain is project.iam.gserviceaccount.com.
	// developer.gserviceaccount.com and appspot.gserviceaccount.com must not be treated as project IDs.
	if !strings.HasSuffix(domain, ".iam.gserviceaccount.com") {
		return ""
	}
	before, _, ok := strings.Cut(domain, ".")
	if !ok {
		return ""
	}
	return before
}

// isGoogleManagedServiceAccountDomain reports whether the domain's first label is a known
// Google-managed pattern (not a customer project ID). Such SAs cannot be resolved to a
// project-scoped IAMServiceAccount item with a valid Scope.
func isGoogleManagedServiceAccountDomain(firstLabel string) bool {
	// gcp-sa-* (e.g. gcp-sa-logging, gcp-sa-datalabeling)
	if strings.HasPrefix(firstLabel, "gcp-sa-") {
		return true
	}
	// cloudservices.gserviceaccount.com, gs-project-accounts, system.gserviceaccount.com
	switch firstLabel {
	case "cloudservices", "gs-project-accounts", "system":
		return true
	}
	return false
}
