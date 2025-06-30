package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		t.Errorf("expected %d linked item query test cases, got %d", len(item.GetLinkedItemQueries()), len(i))
	}

	linkedItemQueries := make(map[string]*sdp.LinkedItemQuery, len(i))
	for _, lir := range item.GetLinkedItemQueries() {
		queryK := queryKey(lir.GetQuery().GetType(), lir.GetQuery().GetQuery())
		if _, ok := linkedItemQueries[queryK]; ok {
			t.Fatalf("linked item query %s for %s already exists in actual linked item queries", lir.GetQuery().GetType(), lir.GetQuery().GetQuery())
		}
		linkedItemQueries[queryK] = lir
	}

	for _, test := range i {
		queryK := queryKey(test.ExpectedType, test.ExpectedQuery)
		gotLiq, ok := linkedItemQueries[queryK]
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

func queryKey(itemType, query string) string {
	return fmt.Sprintf("%s/%s", itemType, query)
}

type mockRoundTripper struct {
	responses map[string]*http.Response
}

func newMockRoundTripper(responses map[string]*http.Response) *mockRoundTripper {
	return &mockRoundTripper{
		responses: responses,
	}
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, ok := m.responses[req.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Not found"}`)),
			Header:     make(http.Header),
		}, nil
	}

	// Clone the response body since it will be closed by the caller
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return resp, nil
}

// mockHTTPResponse converts an input to an io.ReadCloser
// for use in HTTP response mocking
func mockHTTPResponse(input any) io.ReadCloser {
	data, err := json.Marshal(input)
	if err != nil {
		// For test helpers, it's reasonable to panic on marshaling errors
		panic(fmt.Sprintf("Failed to marshal instance input: %v", err))
	}
	return io.NopCloser(bytes.NewReader(data))
}

// MockResponse is a struct that defines the expected response for a mocked HTTP call.
// It includes the status code and the body of the response.
// Body can be any type, but it is typically a struct that can be marshaled to JSON.
type MockResponse struct {
	StatusCode int
	Body       any
}

// NewMockHTTPClientProvider creates a new mock HTTP client provider with the given expected calls and responses.
// The expectedCallAndResponse map should have the URL as the key and a MockResponse as the value.
func NewMockHTTPClientProvider(expectedCallAndResponse map[string]MockResponse) *http.Client {
	cp := make(map[string]*http.Response, len(expectedCallAndResponse))
	for url, resp := range expectedCallAndResponse {
		body := mockHTTPResponse(resp.Body)
		cp[url] = &http.Response{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	return &http.Client{
		Transport: newMockRoundTripper(cp),
	}
}
