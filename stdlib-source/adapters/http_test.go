package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

const TestHTTPTimeout = 3 * time.Second

type TestHTTPServer struct {
	TLSServer               *httptest.Server
	HTTPServer              *httptest.Server
	NotFoundPage            string // A page that returns a 404
	InternalServerErrorPage string // A page that returns a 500
	RedirectPage            string // A page that returns a 301
	RedirectPageRelative    string // A page that returns a 301 with relative location
	RedirectPageLinkLocal   string // A page that returns a 301 redirecting to link-local address
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

	sm.Handle("/301-relative", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "/redirected?param=value#fragment")
		w.WriteHeader(http.StatusMovedPermanently)
	}))

	sm.Handle("/301-link-local", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "http://169.254.169.254/latest/meta-data/")
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
		RedirectPageRelative:    fmt.Sprintf("%v/301-relative", tlsServer.URL),
		RedirectPageLinkLocal:   fmt.Sprintf("%v/301-link-local", tlsServer.URL),
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
	src := HTTPAdapter{
		cache: sdpcache.NewNoOpCache(),
	}
	server, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.TLSServer.Close()

	t.Run("With a specified port and dns name", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", "https://"+net.JoinHostPort("localhost", server.Port), false)
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

	t.Run("With a 301 redirect with relative location", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", server.RedirectPageRelative, false)
		if err != nil {
			t.Fatal(err)
		}

		var status interface{}
		status, err = item.GetAttributes().Get("status")
		if err != nil {
			t.Fatal(err)
		}

		if status != float64(301) {
			t.Errorf("Expected status to be 301, got: %v", status)
		}

		// Check that the location header contains the relative URL
		var location interface{}
		location, err = item.GetAttributes().Get("location")
		if err != nil {
			t.Fatal(err)
		}

		if location != "/redirected?param=value#fragment" {
			t.Errorf("Expected location to be /redirected?param=value#fragment, got: %v", location)
		}

		// Check that the linked item has the resolved absolute URL
		liqs := item.GetLinkedItemQueries()
		if len(liqs) != 3 {
			t.Errorf("expected linked items for redirected location, ip, and dns, got %v: %v", len(liqs), liqs)
		}

		// Extract the base URL from the test server URL
		expectedResolvedURL := "https://" + net.JoinHostPort("127.0.0.1", server.Port) + "/redirected"

		for l := range liqs {
			// Look for the linked item with the http query to the redirect
			// location, check that the relative URL was resolved to absolute.
			if liqs[l].GetQuery().GetType() == "http" {
				if liqs[l].GetQuery().GetQuery() != expectedResolvedURL {
					t.Errorf("expected linked item query to be %s, got %v", expectedResolvedURL, liqs[l].GetQuery().GetQuery())
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

	t.Run("With query parameters should return error", func(t *testing.T) {
		urlWithQuery := server.OKPage + "?param=value"

		_, err := src.Get(context.Background(), "global", urlWithQuery, false)

		if err == nil {
			t.Error("Expected error for URL with query parameters, got nil")
		}
	})

	t.Run("With fragment should return error", func(t *testing.T) {
		urlWithFragment := server.OKPage + "#fragment"

		_, err := src.Get(context.Background(), "global", urlWithFragment, false)

		if err == nil {
			t.Error("Expected error for URL with fragment, got nil")
		}
	})

	t.Run("With both query parameters and fragment should return error", func(t *testing.T) {
		urlWithBoth := server.OKPage + "?param=value#fragment"

		_, err := src.Get(context.Background(), "global", urlWithBoth, false)

		if err == nil {
			t.Error("Expected error for URL with query parameters and fragment, got nil")
		}
	})

	t.Run("With link-local IP address should be blocked", func(t *testing.T) {
		// Test direct access to EC2 metadata service IP
		metadataURL := "http://169.254.169.254/latest/meta-data/"

		_, err := src.Get(context.Background(), "global", metadataURL, false)

		if err == nil {
			t.Error("Expected error for link-local IP address, got nil")
		}

		// Verify the error message mentions link-local blocking
		if err != nil {
			errStr := err.Error()
			if errStr == "" {
				t.Error("Expected error message, got empty string")
			}
			// Check that it's a QueryError with the right error type
			var qErr *sdp.QueryError
			if errors.As(err, &qErr) {
				if qErr.GetErrorType() != sdp.QueryError_OTHER {
					t.Errorf("Expected error type OTHER, got %v", qErr.GetErrorType())
				}
				if !strings.Contains(qErr.GetErrorString(), "link-local") {
					t.Errorf("Expected error message to mention 'link-local', got: %s", qErr.GetErrorString())
				}
			}
		}
	})

	t.Run("With other link-local IP addresses should be blocked", func(t *testing.T) {
		// Test other IPs in the 169.254.0.0/16 range
		testIPs := []string{
			"http://169.254.0.1/",
			"http://169.254.1.1/",
			"http://169.254.255.255/",
		}

		for _, testIP := range testIPs {
			_, err := src.Get(context.Background(), "global", testIP, false)

			if err == nil {
				t.Errorf("Expected error for link-local IP %s, got nil", testIP)
			}
		}
	})

	t.Run("With redirect to link-local address should be blocked", func(t *testing.T) {
		// Test that redirects to link-local addresses are blocked
		item, err := src.Get(context.Background(), "global", server.RedirectPageLinkLocal, false)
		if err != nil {
			t.Fatal(err)
		}

		// The request should succeed, but the redirect should be marked as blocked
		var locationError interface{}
		locationError, err = item.GetAttributes().Get("location-error")
		if err != nil {
			t.Fatal("Expected location-error attribute for blocked redirect")
		}

		locationErrorStr := locationError.(string)
		if !strings.Contains(locationErrorStr, "redirect blocked") {
			t.Errorf("Expected location-error to contain 'redirect blocked', got: %s", locationErrorStr)
		}
		if !strings.Contains(locationErrorStr, "link-local") {
			t.Errorf("Expected location-error to mention 'link-local', got: %s", locationErrorStr)
		}

		// Verify that no linked item query was created for the blocked redirect
		liqs := item.GetLinkedItemQueries()
		for _, liq := range liqs {
			if liq.GetQuery().GetType() == "http" {
				if strings.Contains(liq.GetQuery().GetQuery(), "169.254") {
					t.Errorf("Expected no linked item query for blocked link-local redirect, got: %s", liq.GetQuery().GetQuery())
				}
			}
		}
	})
}

func TestHTTPSearch(t *testing.T) {
	src := HTTPAdapter{
		cache: sdpcache.NewNoOpCache(),
	}
	server, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.TLSServer.Close()

	t.Run("With query parameters and fragments", func(t *testing.T) {
		// Test URL with query parameters and fragments
		testURL := server.OKPage + "?param1=value1&param2=value2#fragment"

		items, err := src.Search(context.Background(), "global", testURL, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(items))
		}

		item := items[0]

		// The unique attribute should be the clean URL without query params and fragments
		expectedCleanURL := server.OKPage
		if item.UniqueAttributeValue() != expectedCleanURL {
			t.Errorf("Expected unique attribute to be %s, got %s", expectedCleanURL, item.UniqueAttributeValue())
		}

		// Verify the item has the expected status (200 for OK page)
		var status interface{}
		status, err = item.GetAttributes().Get("status")
		if err != nil {
			t.Fatal(err)
		}

		if status != float64(200) {
			t.Errorf("Expected status to be 200, got: %v", status)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("With invalid URL", func(t *testing.T) {
		invalidURL := "not-a-valid-url"

		_, err := src.Search(context.Background(), "global", invalidURL, false)

		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("With wrong scope", func(t *testing.T) {
		_, err := src.Search(context.Background(), "wrong-scope", server.OKPage, false)

		if err == nil {
			t.Error("Expected error for wrong scope, got nil")
		}
	})
}
