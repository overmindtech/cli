package adapters

import (
	"context"
	"testing"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdpcache"
)

func TestDomainAdapterGet(t *testing.T) {
	t.Parallel()

	src := &RdapDomainAdapter{
		ClientFac: func() *rdap.Client { return testRdapClient(t) },
		Cache:     sdpcache.NewCache(),
	}

	t.Run("without a dot", func(t *testing.T) {
		items, err := src.Search(context.Background(), "global", "reddit.map.fastly.net", false)

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
	})

	t.Run("with a dot", func(t *testing.T) {
		items, err := src.Search(context.Background(), "global", "reddit.map.fastly.net.", false)

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
	})
}
