package adapters

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/overmindtech/cli/discovery"
)

const TestHTTPTimeout = 3 * time.Second

type TestHTTPServer struct {
	TLSServer               *httptest.Server
	HTTPServer              *httptest.Server
	NotFoundPage            string // A page that returns a 404
	InternalServerErrorPage string // A page that returns a 500
	RedirectPage            string // A page that returns a 301
	SlowPage                string // A page that takes longer than the timeout to respond
	OKPage                  string // A page that returns a 200
	OKPageNoTLS             string // A page that returns a 200 without TLS
	Host                    string
	Port                    string
}

func NewTestServer() (*TestHTTPServer, error) {
	sm := http.NewServeMux()

	sm.Handle("/404", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("not found innit"))
		if err != nil {
			return
		}
	}))

	sm.Handle("/500", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("yeah nah innit"))
		if err != nil {
			return
		}
	}))

	sm.Handle("/301", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "https://www.google.com?foo=bar#baz")
		w.WriteHeader(http.StatusMovedPermanently)
	}))

	sm.Handle("/200", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok innit"))
		if err != nil {
			return
		}
	}))

	sm.Handle("/slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, err := w.Write([]byte("ok innit"))
		if err != nil {
			return
		}
	}))

	tlsServer := httptest.NewTLSServer(sm)
	httpServer := httptest.NewServer(sm)

	host, port, err := net.SplitHostPort(tlsServer.Listener.Addr().String())
	if err != nil {
		return nil, err
	}

	return &TestHTTPServer{
		TLSServer:               tlsServer,
		HTTPServer:              httpServer,
		NotFoundPage:            fmt.Sprintf("%v/404", tlsServer.URL),
		InternalServerErrorPage: fmt.Sprintf("%v/500", tlsServer.URL),
		RedirectPage:            fmt.Sprintf("%v/301", tlsServer.URL),
		OKPage:                  fmt.Sprintf("%v/200", tlsServer.URL),
		OKPageNoTLS:             fmt.Sprintf("%v/200", httpServer.URL),
		SlowPage:                fmt.Sprintf("%v/slow", tlsServer.URL),
		Host:                    host,
		Port:                    port,
	}, nil
}

func (t *TestHTTPServer) Close() {
	if t.TLSServer != nil {
		t.TLSServer.Close()
	}
	if t.HTTPServer != nil {
		t.HTTPServer.Close()
	}
}

func TestHTTPGet(t *testing.T) {
	src := HTTPAdapter{}
	server, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.TLSServer.Close()

	t.Run("With a specified port and dns name", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", fmt.Sprintf("https://localhost:%v", server.Port), false)

		if err != nil {
			t.Fatal(err)
		}

		var dnsFound bool

		for _, link := range item.GetLinkedItemQueries() {
			switch link.GetQuery().GetType() {
			case "dns":
				dnsFound = true

				if link.GetQuery().GetQuery() != "localhost" {
					t.Errorf("expected dns query to be localhost, got %v", link.GetQuery())
				}
			}
		}

		if !dnsFound {
			t.Error("link to dns not found")
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With an IP", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.OKPage, false)

		if err != nil {
			t.Fatal(err)
		}

		var ipFound bool

		for _, link := range item.GetLinkedItemQueries() {
			switch link.GetQuery().GetType() {
			case "ip":
				ipFound = true

				if link.GetQuery().GetQuery() != "127.0.0.1" {
					t.Errorf("expected dns query to be 127.0.0.1, got %v", link.GetQuery())
				}
			}
		}

		if !ipFound {
			t.Error("link to ip not found")
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With a 404", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.NotFoundPage, false)

		if err != nil {
			t.Fatal(err)
		}

		var status interface{}

		status, err = item.GetAttributes().Get("status")

		if err != nil {
			t.Fatal(err)
		}

		if status != float64(404) {
			t.Errorf("expected status to be 404, got: %v", status)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With a timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		item, err := src.Get(ctx, "global", server.SlowPage, false)

		if err == nil {
			t.Errorf("Expected timeout but got %v", item.String())
		}
	})

	t.Run("With a 500 error", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.InternalServerErrorPage, false)

		if err != nil {
			t.Fatal(err)
		}

		var status interface{}

		status, err = item.GetAttributes().Get("status")

		if err != nil {
			t.Fatal(err)
		}

		if status != float64(500) {
			t.Errorf("expected status to be 500, got: %v", status)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With a 301 redirect", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.RedirectPage, false)

		if err != nil {
			t.Fatal(err)
		}

		var status interface{}

		status, err = item.GetAttributes().Get("status")

		if err != nil {
			t.Fatal(err)
		}

		if status != float64(301) {
			t.Errorf("expected status to be 301, got: %v", status)
		}

		liqs := item.GetLinkedItemQueries()
		if len(liqs) != 3 {
			t.Errorf("expected linked items for redirected location, ip, and dns, got %v: %v", len(liqs), liqs)
		}
		for l := range liqs {
			// Look for the linked item with the http query to the redirect
			// location, check that the query and fragment have been stripped.
			if liqs[l].GetQuery().GetType() == "http" {
				if liqs[l].GetQuery().GetQuery() != "https://www.google.com" {
					t.Errorf("expected linked item query to be https://www.google.com, got %v", liqs[l].GetQuery().GetQuery())
				}
			}
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With no TLS", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.OKPageNoTLS, false)

		if err != nil {
			t.Fatal(err)
		}

		_, err = item.GetAttributes().Get("tls")

		if err == nil {
			t.Error("Expected to not find TLS info")
		}

		discovery.TestValidateItem(t, item)
	})
}
