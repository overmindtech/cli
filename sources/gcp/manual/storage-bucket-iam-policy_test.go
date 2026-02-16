package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// fakeBucketIAMPolicyGetter returns a fixed list of bindings for testing.
type fakeBucketIAMPolicyGetter struct {
	bindings   []gcpshared.BucketIAMBinding
	returnErr  error
	bucketSeen string
}

func (f *fakeBucketIAMPolicyGetter) GetBucketIAMPolicy(ctx context.Context, bucketName string) ([]gcpshared.BucketIAMBinding, error) {
	f.bucketSeen = bucketName
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	return f.bindings, nil
}

// policyWithBindings builds []BucketIAMBinding from role -> members (no condition).
// For conditional bindings, construct []BucketIAMBinding directly.
func policyWithBindings(bindings map[string][]string) []gcpshared.BucketIAMBinding {
	out := make([]gcpshared.BucketIAMBinding, 0, len(bindings))
	for role, members := range bindings {
		out = append(out, gcpshared.BucketIAMBinding{Role: role, Members: members, ConditionExpression: ""})
	}
	return out
}

func TestStorageBucketIAMPolicy_Get(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	bucketName := "my-bucket"
	role := "roles/storage.objectViewer"
	saMember := "serviceAccount:siem-sa@test-project.iam.gserviceaccount.com"

	bindings := policyWithBindings(map[string][]string{
		role: {saMember, "user:alice@example.com"},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	scope := projectID
	sdpItem, qErr := adapter.Get(ctx, scope, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	if sdpItem.GetType() != gcpshared.StorageBucketIAMPolicy.String() {
		t.Errorf("type: got %s, want %s", sdpItem.GetType(), gcpshared.StorageBucketIAMPolicy.String())
	}
	if getter.bucketSeen != bucketName {
		t.Errorf("bucket seen: got %s, want %s", getter.bucketSeen, bucketName)
	}

	// Policy item has bucket and bindings attributes
	if ua, _ := sdpItem.GetAttributes().Get("uniqueAttr"); ua != bucketName {
		t.Errorf("uniqueAttr: got %v, want %s", ua, bucketName)
	}

	t.Run("StaticTests", func(t *testing.T) {
		queryTests := shared.QueryTests{
			{
				ExpectedType:             gcpshared.StorageBucket.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            bucketName,
				ExpectedScope:            projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			},
			{
				ExpectedType:             gcpshared.IAMServiceAccount.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "siem-sa@test-project.iam.gserviceaccount.com",
				ExpectedScope:            projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			},
		}
		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})
}

func TestStorageBucketIAMPolicy_Get_ProjectPrincipalMembers_Linked(t *testing.T) {
	ctx := context.Background()
	projectID := "bucket-project"
	bucketName := "my-bucket"
	role := "roles/storage.objectViewer"
	bindings := policyWithBindings(map[string][]string{
		role: {
			"projectOwner:other-project",
			"projectEditor:another-project",
			"projectViewer:bucket-project",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	t.Run("StaticTests", func(t *testing.T) {
		queryTests := shared.QueryTests{
			{
				ExpectedType:             gcpshared.StorageBucket.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            bucketName,
				ExpectedScope:            projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			},
			{
				ExpectedType:             gcpshared.ComputeProject.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "other-project",
				ExpectedScope:            "other-project",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			},
			{
				ExpectedType:             gcpshared.ComputeProject.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "another-project",
				ExpectedScope:            "another-project",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			},
			{
				ExpectedType:             gcpshared.ComputeProject.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "bucket-project",
				ExpectedScope:            "bucket-project",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			},
		}
		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})
}

func TestStorageBucketIAMPolicy_Get_ProjectPrincipalMembers_Deduplicated(t *testing.T) {
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.admin": {
			"projectOwner:shared-project",
			"projectEditor:shared-project",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var projectLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.ComputeProject.String() {
			projectLinks++
			if q.GetQuery().GetQuery() != "shared-project" || q.GetQuery().GetScope() != "shared-project" {
				t.Errorf("ComputeProject link: got query=%q scope=%q, want shared-project", q.GetQuery().GetQuery(), q.GetQuery().GetScope())
			}
		}
	}
	if projectLinks != 1 {
		t.Errorf("expected 1 ComputeProject link (deduplicated), got %d", projectLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_DeletedServiceAccount_IsLinked(t *testing.T) {
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {
			"deleted:serviceAccount:old-sa@my-project.iam.gserviceaccount.com?uid=123456789",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var iamLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.IAMServiceAccount.String() {
			iamLinks++
			if q.GetQuery().GetScope() != "my-project" {
				t.Errorf("IAM link scope: got %q, want my-project", q.GetQuery().GetScope())
			}
		}
	}
	if iamLinks != 1 {
		t.Errorf("expected 1 IAMServiceAccount link for deleted:serviceAccount: member, got %d", iamLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_DomainMembers_EmitDNSLinks(t *testing.T) {
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {
			"domain:example.com",
			"domain:acme.co.uk",
			"domain:example.com",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var dnsLinks int
	dnsQueries := make(map[string]struct{})
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == "dns" {
			dnsLinks++
			dnsQueries[q.GetQuery().GetQuery()] = struct{}{}
			if q.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH || q.GetQuery().GetScope() != "global" {
				t.Errorf("dns link: method=%v scope=%q (want SEARCH, global)", q.GetQuery().GetMethod(), q.GetQuery().GetScope())
			}
		}
	}
	if dnsLinks != 2 {
		t.Errorf("expected 2 dns links (example.com, acme.co.uk; example.com deduped), got %d", dnsLinks)
	}
	if _, ok := dnsQueries["example.com"]; !ok {
		t.Error("missing dns link for example.com")
	}
	if _, ok := dnsQueries["acme.co.uk"]; !ok {
		t.Error("missing dns link for acme.co.uk")
	}
}

func TestStorageBucketIAMPolicy_Get_DeletedDomainMember_StripsUIDSuffix(t *testing.T) {
	// deleted:domain:example.com?uid=123456789 should produce a DNS link with query "example.com", not "example.com?uid=123456789".
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {
			"deleted:domain:example.com?uid=123456789",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var dnsLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == "dns" {
			dnsLinks++
			query := q.GetQuery().GetQuery()
			if query != "example.com" {
				t.Errorf("dns link query: got %q, want example.com (?uid= suffix must be stripped)", query)
			}
		}
	}
	if dnsLinks != 1 {
		t.Errorf("expected 1 dns link, got %d", dnsLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_CustomRole_EmitsIAMRoleLink(t *testing.T) {
	// Bindings that reference custom IAM roles (projects/{project}/roles/{roleId}) should emit LinkedItemQuery to IAMRole.
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := []gcpshared.BucketIAMBinding{
		{
			Role:                 "projects/custom-project/roles/myCustomRole",
			Members:              []string{"user:admin@example.com"},
			ConditionExpression:  "",
			ConditionTitle:       "",
			ConditionDescription: "",
		},
		{
			Role:                 "roles/storage.objectViewer",
			Members:              []string{"user:viewer@example.com"},
			ConditionExpression:  "",
			ConditionTitle:       "",
			ConditionDescription: "",
		},
	}
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var iamRoleLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.IAMRole.String() {
			iamRoleLinks++
			if q.GetQuery().GetScope() != "custom-project" || q.GetQuery().GetQuery() != "myCustomRole" {
				t.Errorf("IAMRole link: got scope=%q query=%q, want scope=custom-project query=myCustomRole", q.GetQuery().GetScope(), q.GetQuery().GetQuery())
			}
		}
	}
	if iamRoleLinks != 1 {
		t.Errorf("expected 1 IAMRole link for custom role, got %d", iamRoleLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_GoogleManagedSA_SkipsLink(t *testing.T) {
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {
			"serviceAccount:my-sa@my-project.iam.gserviceaccount.com",
			"serviceAccount:123456@gcp-sa-logging.iam.gserviceaccount.com",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var iamLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.IAMServiceAccount.String() {
			iamLinks++
			if q.GetQuery().GetScope() != "my-project" || q.GetQuery().GetQuery() != "my-sa@my-project.iam.gserviceaccount.com" {
				t.Errorf("IAM link: scope=%q query=%q (expected customer SA only)", q.GetQuery().GetScope(), q.GetQuery().GetQuery())
			}
		}
	}
	if iamLinks != 1 {
		t.Errorf("expected 1 IAMServiceAccount link (customer SA), got %d (Google-managed SA should be skipped)", iamLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_DeveloperAndAppspotSA_SkipLink(t *testing.T) {
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {
			"serviceAccount:123456@developer.gserviceaccount.com",
			"serviceAccount:my-app@appspot.gserviceaccount.com",
		},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed: %v", qErr)
		return
	}

	var iamLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.IAMServiceAccount.String() {
			iamLinks++
			scope := q.GetQuery().GetScope()
			if scope == "developer" || scope == "appspot" {
				t.Errorf("must not create IAM link with scope %q (not a project ID)", scope)
			}
		}
	}
	if iamLinks != 0 {
		t.Errorf("expected 0 IAMServiceAccount links for developer/appspot SAs, got %d", iamLinks)
	}
}

func TestStorageBucketIAMPolicy_Get_ClientError(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	getter := &fakeBucketIAMPolicyGetter{returnErr: errors.New("api error"), bindings: nil}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	_, qErr := adapter.Get(ctx, projectID, "my-bucket", true)
	if qErr == nil {
		t.Error("expected error when getter returns error")
		return
	}
}

func TestStorageBucketIAMPolicy_Search(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	bucketName := "my-bucket"
	bindings := policyWithBindings(map[string][]string{
		"roles/storage.objectViewer": {"serviceAccount:sa1@test-project.iam.gserviceaccount.com"},
		"roles/storage.admin":        {"user:admin@example.com"},
	})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	searchable, ok := adapter.(discovery.SearchableAdapter)
	if !ok {
		t.Error("adapter does not implement SearchableAdapter")
		return
	}

	items, qErr := searchable.Search(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Search failed: %v", qErr)
		return
	}

	if len(items) != 1 {
		t.Errorf("Search: got %d items, want 1 (one policy per bucket)", len(items))
	}
	if getter.bucketSeen != bucketName {
		t.Errorf("bucket seen: got %s, want %s", getter.bucketSeen, bucketName)
	}

	if len(items) > 0 {
		if err := items[0].Validate(); err != nil {
			t.Errorf("item validation: %v", err)
		}
		if items[0].GetType() != gcpshared.StorageBucketIAMPolicy.String() {
			t.Errorf("Search item type: got %s, want %s", items[0].GetType(), gcpshared.StorageBucketIAMPolicy.String())
		}
	}
}

func TestStorageBucketIAMPolicy_TerraformMapping(t *testing.T) {
	bindings := policyWithBindings(map[string][]string{"roles/storage.objectViewer": {"user:u@example.com"}})
	getter := &fakeBucketIAMPolicyGetter{bindings: bindings}
	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation("p")})

	mappings := wrapper.TerraformMappings()
	wantMaps := map[string]bool{
		"google_storage_bucket_iam_binding.bucket": false,
		"google_storage_bucket_iam_member.bucket":  false,
		"google_storage_bucket_iam_policy.bucket":  false,
	}
	if len(mappings) != 3 {
		t.Errorf("TerraformMappings: got %d entries, want 3", len(mappings))
		return
	}
	for _, m := range mappings {
		if m.GetTerraformMethod() != sdp.QueryMethod_GET {
			t.Errorf("TerraformMethod: got %v, want GET", m.GetTerraformMethod())
		}
		qm := m.GetTerraformQueryMap()
		if _, ok := wantMaps[qm]; !ok {
			t.Errorf("TerraformQueryMap: unexpected %q", qm)
		}
		wantMaps[qm] = true
	}
	for qm, seen := range wantMaps {
		if !seen {
			t.Errorf("TerraformQueryMap: missing %q", qm)
		}
	}
}

func TestStorageBucketIAMPolicy_Get_InsufficientQueryParts(t *testing.T) {
	ctx := context.Background()
	getter := &fakeBucketIAMPolicyGetter{bindings: nil}
	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation("p")})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	// Get with empty query should fail (no bucket name)
	_, qErr := adapter.Get(ctx, "p", "", true)
	if qErr == nil {
		t.Error("expected error when query is empty (no bucket name)")
		return
	}
}

func TestStorageBucketIAMPolicy_Get_EmptyPolicy_ReturnsItem(t *testing.T) {
	// Bucket with no bindings still returns a valid policy item (empty bindings array).
	ctx := context.Background()
	projectID := "my-project"
	bucketName := "my-bucket"
	getter := &fakeBucketIAMPolicyGetter{bindings: []gcpshared.BucketIAMBinding{}}

	wrapper := manual.NewStorageBucketIAMPolicy(getter, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
	adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

	sdpItem, qErr := adapter.Get(ctx, projectID, bucketName, true)
	if qErr != nil {
		t.Errorf("Get failed for empty policy: %v", qErr)
		return
	}
	if sdpItem.GetType() != gcpshared.StorageBucketIAMPolicy.String() {
		t.Errorf("type: got %s, want %s", sdpItem.GetType(), gcpshared.StorageBucketIAMPolicy.String())
	}
	// Should still link to the bucket
	var bucketLinks int
	for _, q := range sdpItem.GetLinkedItemQueries() {
		if q.GetQuery().GetType() == gcpshared.StorageBucket.String() {
			bucketLinks++
		}
	}
	if bucketLinks != 1 {
		t.Errorf("expected 1 StorageBucket link, got %d", bucketLinks)
	}
}
