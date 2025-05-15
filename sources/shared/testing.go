package shared

import (
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

// RunStaticTests runs static tests on the given adapter and item.
// It validates the adapter and item, and runs the provided query tests for linked items and potential links.
func RunStaticTests(t *testing.T, adapter discovery.Adapter, item *sdp.Item, queryTests QueryTests) {
	if adapter == nil {
		t.Fatal("adapter is nil")
	}

	ValidateAdapter(t, adapter)

	if item == nil {
		t.Fatal("item is nil")
	}

	if item.Validate() != nil {
		t.Fatalf("Item %s failed validation: %v", item.GetType(), item.Validate())
	}

	if queryTests == nil {
		t.Skipf("Skipping test because no query test provided")
	}

	queryTests.Execute(t, item, adapter)
}

type Validate interface {
	Validate() error
}

func ValidateAdapter(t *testing.T, adapter discovery.Adapter) {
	if adapter == nil {
		t.Fatal("adapter is nil")
	}

	// Test the adapter
	a, ok := adapter.(Validate)
	if !ok {
		t.Fatalf("Adapter %s does not implement Validate", adapter.Name())
	}

	if err := a.Validate(); err != nil {
		t.Fatalf("Adapter %s failed validation: %v", adapter.Name(), err)
	}
}

// QueryTest is a struct that defines the expected properties of a linked item query.
type QueryTest struct {
	ExpectedType             string
	ExpectedMethod           sdp.QueryMethod
	ExpectedQuery            string
	ExpectedScope            string
	ExpectedBlastPropagation *sdp.BlastPropagation
}

type QueryTests []QueryTest

// TestLinkedItems tests the linked item queries of an item for the expected properties.
func (i QueryTests) TestLinkedItems(t *testing.T, item *sdp.Item) {
	if item == nil {
		t.Fatal("item is nil")
	}

	if item.GetLinkedItemQueries() == nil {
		t.Fatal("item.GetLinkedItemQueries() is nil")
	}

	if len(i) != len(item.GetLinkedItemQueries()) {
		t.Fatalf("expected %d linked item query test cases, got %d", len(item.GetLinkedItemQueries()), len(i))
	}

	linkedItemQueries := make(map[string]*sdp.LinkedItemQuery, len(i))
	for _, lir := range item.GetLinkedItemQueries() {
		linkedItemQueries[lir.GetQuery().GetQuery()] = lir
	}

	for _, test := range i {
		gotLiq, ok := linkedItemQueries[test.ExpectedQuery]
		if !ok {
			t.Fatalf("linked item query %s for %s not found in actual linked item queries", test.ExpectedType, test.ExpectedQuery)
		}

		if test.ExpectedScope != gotLiq.GetQuery().GetScope() {
			t.Errorf("for the linked item query %s of %s, expected scope %s, got %s", test.ExpectedQuery, test.ExpectedType, test.ExpectedScope, gotLiq.GetQuery().GetScope())
		}

		if test.ExpectedType != gotLiq.GetQuery().GetType() {
			t.Errorf("for the linked item query %s, expected type %s, got %s", test.ExpectedQuery, test.ExpectedType, gotLiq.GetQuery().GetType())
		}

		if test.ExpectedMethod != gotLiq.GetQuery().GetMethod() {
			t.Errorf("for the linked item query %s of %s, expected method %s, got %s", test.ExpectedQuery, test.ExpectedType, test.ExpectedMethod, gotLiq.GetQuery().GetMethod())
		}

		if test.ExpectedBlastPropagation == nil {
			t.Fatalf("for the linked item query %s of %s, the test case must have a non-nil blast propagation", test.ExpectedQuery, test.ExpectedType)
		}

		if gotLiq.GetBlastPropagation() == nil {
			t.Fatalf("for the linked item query %s of %s, expected blast propagation to be non-nil", test.ExpectedQuery, test.ExpectedType)
		}

		if test.ExpectedBlastPropagation.GetIn() != gotLiq.GetBlastPropagation().GetIn() {
			t.Errorf("for the linked item query %s of %s, expected blast propagation [IN] to be %v, got %v", test.ExpectedQuery, test.ExpectedType, test.ExpectedBlastPropagation.GetIn(), gotLiq.GetBlastPropagation().GetIn())
		}

		if test.ExpectedBlastPropagation.GetOut() != gotLiq.GetBlastPropagation().GetOut() {
			t.Errorf("for the linked item query %s of %s, expected blast propagation [OUT] to be %v, got %v", test.ExpectedQuery, test.ExpectedType, test.ExpectedBlastPropagation.GetOut(), gotLiq.GetBlastPropagation().GetOut())
		}
	}
}

// TestPotentialLinks tests the potential links of an adapter for the given item.
func (i QueryTests) TestPotentialLinks(t *testing.T, item *sdp.Item, adapter discovery.Adapter) {
	if adapter == nil {
		t.Fatal("adapter is nil")
	}

	if adapter.Metadata() == nil {
		t.Fatal("adapter.Metadata() is nil")
	}

	if adapter.Metadata().GetPotentialLinks() == nil {
		t.Fatal("adapter.Metadata().GetPotentialLinks() is nil")
	}

	potentialLinks := make(map[string]bool, len(i))
	for _, l := range adapter.Metadata().GetPotentialLinks() {
		potentialLinks[l] = true
	}

	if item == nil {
		t.Fatal("item is nil")
	}

	for _, test := range i {
		if _, ok := potentialLinks[test.ExpectedType]; !ok {
			t.Fatalf("linked item type %s not found in potential links", test.ExpectedType)
		}
	}
}

func (i QueryTests) Execute(t *testing.T, item *sdp.Item, adapter discovery.Adapter) {
	t.Run("LinkedItemQueries", func(t *testing.T) {
		i.TestLinkedItems(t, item)
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		i.TestPotentialLinks(t, item, adapter)
	})
}
