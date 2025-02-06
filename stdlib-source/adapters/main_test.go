package adapters

import (
	"net/http"
	"testing"

	"github.com/openrdap/rdap"
	"github.com/openrdap/rdap/bootstrap"
)

func testRdapClient(t *testing.T) *rdap.Client {
	return &rdap.Client{
		HTTP: http.DefaultClient,
		Bootstrap: &bootstrap.Client{
			Verbose: func(text string) {
				t.Log(text)
			},
		},
		Verbose: func(text string) {
			t.Log(text)
		},
	}
}
