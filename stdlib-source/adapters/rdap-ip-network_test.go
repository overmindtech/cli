package adapters

import (
	"context"
	"testing"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdpcache"
)

func TestIpNetworkAdapterSearch(t *testing.T) {
	t.Parallel()

	src := &RdapIPNetworkAdapter{
		ClientFac: func() *rdap.Client { return testRdapClient(t) },
		Cache:     sdpcache.NewCache(),
		IPCache:   NewIPCache[*rdap.IPNetwork](),
	}

	items, err := src.Search(context.Background(), "global", "1.1.1.1", false)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %v", len(items))
	}

	item := items[0]

	if item.UniqueAttributeValue() != "1.1.1.0 - 1.1.1.255" {
		t.Errorf("Expected unique attribute value to be 1.1.1.0 - 1.1.1.0 - 1.1.1.255, got %v", item.UniqueAttributeValue())
	}

	if len(item.GetLinkedItemQueries()) != 3 {
		t.Errorf("Expected 3 linked items, got %v", len(item.GetLinkedItemQueries()))
	}

	// Then run a get for that same thing and hit the cache
	_, err = src.Get(context.Background(), "global", item.UniqueAttributeValue(), false)

	if err != nil {
		t.Fatal(err)
	}
}

func TestCalculateNetwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Start    string
		End      string
		Expected string
	}{
		{
			Start:    "10.0.0.0",
			End:      "10.0.0.255",
			Expected: "10.0.0.0/24",
		},
		{
			Start:    "10.0.0.0",
			End:      "10.0.0.7",
			Expected: "10.0.0.0/29",
		},
	}

	for _, test := range tests {
		network, err := calculateNetwork(test.Start, test.End)

		if err != nil {
			t.Fatal(err)
		}

		if network.String() != test.Expected {
			t.Errorf("Expected network to be %v, got %v", test.Expected, network.String())
		}
	}
}
