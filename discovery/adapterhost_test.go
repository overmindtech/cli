package discovery

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

func TestAdapterHostExpandQuery(t *testing.T) {
	sh := NewAdapterHost()

	err := sh.AddAdapters(
		&TestAdapter{
			ReturnScopes: []string{"test"},
			ReturnType:   "person",
			ReturnName:   "person",
		},
		&TestAdapter{
			ReturnScopes: []string{"test"},
			ReturnType:   "fish",
			ReturnName:   "fish",
		},
		&TestAdapter{
			ReturnScopes: []string{
				"multiA",
				"multiB",
			},
			ReturnType: "chair",
			ReturnName: "chair",
		},
		&TestAdapter{
			ReturnScopes: []string{"test"},
			ReturnType:   "hidden_person",
			IsHidden:     true,
			ReturnName:   "hidden_person",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Right type wrong scope", func(t *testing.T) {
		req := sdp.Query{
			Type:  "person",
			Scope: "wrong",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 0 {
			t.Fatalf("Expected 0 queries, got %v", len(m))
		}
	})

	t.Run("Right scope wrong type", func(t *testing.T) {
		req := sdp.Query{
			Type:  "wrong",
			Scope: "test",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 0 {
			t.Fatalf("Expected 0 queries, got %v", len(m))
		}
	})

	t.Run("Right both", func(t *testing.T) {
		req := sdp.Query{
			Type:  "person",
			Scope: "test",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 1 {
			t.Fatalf("Expected 1 query, got %v", len(m))
		}
	})

	t.Run("Multi-scope", func(t *testing.T) {
		req := sdp.Query{
			Type:  "chair",
			Scope: "multiB",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 1 {
			t.Fatalf("Expected 1 query, got %v", len(m))
		}
	})

	t.Run("Wildcard scope", func(t *testing.T) {
		req := sdp.Query{
			Type:  "person",
			Scope: sdp.WILDCARD,
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 1 {
			t.Fatalf("Expected 1 query, got %v", len(m))
		}

		req = sdp.Query{
			Type:  "chair",
			Scope: sdp.WILDCARD,
		}

		m = sh.ExpandQuery(&req)

		if len(m) != 2 {
			t.Fatalf("Expected 2 queries, got %v", len(m))
		}
	})

	t.Run("Wildcard type", func(t *testing.T) {
		req := sdp.Query{
			Type:  sdp.WILDCARD,
			Scope: "test",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 2 {
			t.Fatalf("Expected 2 adapters, got %v", len(m))
		}
	})

	t.Run("Wildcard both", func(t *testing.T) {
		req := sdp.Query{
			Type:  sdp.WILDCARD,
			Scope: sdp.WILDCARD,
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 4 {
			t.Fatalf("Expected 4 adapters, got %v", len(m))
		}
	})

	t.Run("substring match", func(t *testing.T) {
		req := sdp.Query{
			Type:  sdp.WILDCARD,
			Scope: "multi",
		}

		m := sh.ExpandQuery(&req)

		if len(m) != 2 {
			t.Fatalf("Expected 2 queries, got %v", len(m))
		}
	})

	t.Run("Listing hidden adapter with wildcard scope", func(t *testing.T) {
		req := sdp.Query{
			Type:  "hidden_person",
			Scope: sdp.WILDCARD,
		}
		if x := len(sh.ExpandQuery(&req)); x != 0 {
			t.Errorf("expected to find 0 adapters, found %v", x)
		}

		req = sdp.Query{
			Type:  "hidden_person",
			Scope: "test",
		}
		if x := len(sh.ExpandQuery(&req)); x != 1 {
			t.Errorf("expected to find 1 adapter, found %v", x)
		}
	})
}

func TestAdapterHostAddAdapters(t *testing.T) {
	sh := NewAdapterHost()

	adapter := TestAdapter{}

	err := sh.AddAdapters(&adapter)
	if err != nil {
		t.Fatal(err)
	}

	if x := len(sh.Adapters()); x != 1 {
		t.Fatalf("Expected 1 adapters, got %v", x)
	}
}

func TestAdapterHostExpandQuery_WildcardScope(t *testing.T) {
	sh := NewAdapterHost()

	// Add regular adapter without wildcard support
	regularAdapter := &TestAdapter{
		ReturnScopes: []string{"project.zone-a", "project.zone-b"},
		ReturnType:   "regular-type",
		ReturnName:   "regular",
	}

	// Add wildcard-supporting adapter
	wildcardAdapter := &TestWildcardAdapter{
		TestAdapter: TestAdapter{
			ReturnScopes: []string{"project.zone-a", "project.zone-b"},
			ReturnType:   "wildcard-type",
			ReturnName:   "wildcard",
		},
		supportsWildcard: true,
	}

	err := sh.AddAdapters(regularAdapter, wildcardAdapter)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Regular adapter with wildcard scope expands to all scopes", func(t *testing.T) {
		req := sdp.Query{
			Type:  "regular-type",
			Scope: sdp.WILDCARD,
		}

		expanded := sh.ExpandQuery(&req)

		// Should expand to 2 queries (one per zone)
		if len(expanded) != 2 {
			t.Fatalf("Expected 2 expanded queries for regular adapter, got %v", len(expanded))
		}

		// Check that scopes are specific, not wildcard
		for q := range expanded {
			if q.GetScope() == sdp.WILDCARD {
				t.Errorf("Expected specific scope, got wildcard")
			}
		}
	})

	t.Run("Wildcard-supporting adapter with wildcard scope does not expand for LIST", func(t *testing.T) {
		req := sdp.Query{
			Type:   "wildcard-type",
			Method: sdp.QueryMethod_LIST,
			Scope:  sdp.WILDCARD,
		}

		expanded := sh.ExpandQuery(&req)

		// Should NOT expand - just 1 query with wildcard scope
		if len(expanded) != 1 {
			t.Fatalf("Expected 1 query for wildcard adapter, got %v", len(expanded))
		}

		// Check that scope is still wildcard
		for q := range expanded {
			if q.GetScope() != sdp.WILDCARD {
				t.Errorf("Expected wildcard scope to be preserved, got %v", q.GetScope())
			}
		}
	})

	t.Run("Wildcard-supporting adapter with wildcard scope expands for GET", func(t *testing.T) {
		req := sdp.Query{
			Type:   "wildcard-type",
			Method: sdp.QueryMethod_GET,
			Scope:  sdp.WILDCARD,
		}

		expanded := sh.ExpandQuery(&req)

		// Should expand to 2 queries (one per scope) for GET
		if len(expanded) != 2 {
			t.Fatalf("Expected 2 expanded queries for wildcard adapter with GET, got %v", len(expanded))
		}

		// Check that scopes are specific, not wildcard
		for q := range expanded {
			if q.GetScope() == sdp.WILDCARD {
				t.Errorf("Expected specific scope for GET, got wildcard")
			}
		}
	})

	t.Run("Wildcard-supporting adapter with wildcard scope expands for SEARCH", func(t *testing.T) {
		req := sdp.Query{
			Type:   "wildcard-type",
			Method: sdp.QueryMethod_SEARCH,
			Scope:  sdp.WILDCARD,
		}

		expanded := sh.ExpandQuery(&req)

		// Should expand to 2 queries (one per scope) for SEARCH
		if len(expanded) != 2 {
			t.Fatalf("Expected 2 expanded queries for wildcard adapter with SEARCH, got %v", len(expanded))
		}

		// Check that scopes are specific, not wildcard
		for q := range expanded {
			if q.GetScope() == sdp.WILDCARD {
				t.Errorf("Expected specific scope for SEARCH, got wildcard")
			}
		}
	})

	t.Run("Wildcard-supporting adapter with specific scope works normally", func(t *testing.T) {
		req := sdp.Query{
			Type:  "wildcard-type",
			Scope: "project.zone-a",
		}

		expanded := sh.ExpandQuery(&req)

		// Should return 1 query with specific scope
		if len(expanded) != 1 {
			t.Fatalf("Expected 1 query, got %v", len(expanded))
		}

		for q := range expanded {
			if q.GetScope() != "project.zone-a" {
				t.Errorf("Expected scope 'project.zone-a', got %v", q.GetScope())
			}
		}
	})

	t.Run("Hidden wildcard adapter with wildcard scope is not included", func(t *testing.T) {
		hiddenWildcardAdapter := &TestWildcardAdapter{
			TestAdapter: TestAdapter{
				ReturnScopes: []string{"project.zone-a"},
				ReturnType:   "hidden-wildcard-type",
				ReturnName:   "hidden-wildcard",
				IsHidden:     true,
			},
			supportsWildcard: true,
		}

		err := sh.AddAdapters(hiddenWildcardAdapter)
		if err != nil {
			t.Fatal(err)
		}

		req := sdp.Query{
			Type:  "hidden-wildcard-type",
			Scope: sdp.WILDCARD,
		}

		expanded := sh.ExpandQuery(&req)

		// Hidden adapters should not be expanded for wildcard scopes
		if len(expanded) != 0 {
			t.Fatalf("Expected 0 queries for hidden wildcard adapter, got %v", len(expanded))
		}
	})
}

// TestWildcardAdapter extends TestAdapter to implement WildcardScopeAdapter
type TestWildcardAdapter struct {
	TestAdapter
	supportsWildcard bool
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
func (t *TestWildcardAdapter) SupportsWildcardScope() bool {
	return t.supportsWildcard
}
