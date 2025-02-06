package adapters

import (
	"context"
	"testing"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdpcache"
)

func TestNameserverAdapterSearch(t *testing.T) {
	t.Parallel()

	src := &RdapNameserverAdapter{
		ClientFac: func() *rdap.Client { return testRdapClient(t) },
		Cache:     sdpcache.NewCache(),
	}

	items, err := src.Search(context.Background(), "global", "https://rdap.verisign.com/com/v1/nameserver/NS4.GOOGLE.COM", false)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatal("Expected 1 item")
	}

	item := items[0]

	err = item.Validate()

	if err != nil {
		t.Error(err)
	}
}
