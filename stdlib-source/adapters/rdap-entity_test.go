package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestEntityAdapterSearch(t *testing.T) {
	t.Parallel()

	realUrls := []string{
		"https://rdap.apnic.net/entity/AIC3-AP",
		"https://rdap.apnic.net/entity/IRT-APNICRANDNET-AU",
		"https://rdap.arin.net/registry/entity/HPINC-Z",
	}

	src := &RdapEntityAdapter{
		ClientFac: func() *rdap.Client { return testRdapClient(t) },
		Cache:     sdpcache.NewCache(),
	}

	for _, realUrl := range realUrls {
		t.Run(realUrl, func(t *testing.T) {
			items, err := src.Search(context.Background(), "global", realUrl, false)

			if err != nil {
				t.Fatal(err)
			}

			if len(items) != 1 {
				t.Fatalf("Expected 1 item, got %v", len(items))
			}

			item := items[0]

			err = item.Validate()

			if err != nil {
				t.Error(err)
			}
		})
	}

	t.Run("not found", func(t *testing.T) {
		_, err := src.Search(context.Background(), "global", "https://rdap.apnic.net/entity/NOTFOUND", false)

		if err == nil {
			t.Fatal("Expected error")
		}

		var sdpError *sdp.QueryError

		if ok := errors.As(err, &sdpError); ok {
			if sdpError.GetErrorType() != sdp.QueryError_NOTFOUND {
				t.Errorf("Expected QueryError_NOTFOUND, got %v", sdpError.GetErrorType())
			}
		} else {
			t.Fatalf("Expected QueryError, got %T", err)
		}
	})
}
