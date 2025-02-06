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
