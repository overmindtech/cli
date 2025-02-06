package adapters

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func TestSearch(t *testing.T) {
	t.Parallel()

	s := DNSAdapter{
		Servers: []string{
			"1.1.1.1:53",
			"8.8.8.8:53",
		},
	}

	t.Run("with a bad DNS name", func(t *testing.T) {
		_, err := s.Search(context.Background(), "global", "not.real.overmind.tech", false)

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("with one.one.one.one", func(t *testing.T) {
		items, err := s.Search(context.Background(), "global", "one.one.one.one", false)

		if err != nil {
			t.Error(err)
		}

		if len(items) != 1 {
			t.Errorf("expected 1 item, got %v", len(items))
		}

		// Make sure 1.1.1.1 is in there
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == "1.1.1.1" {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == "2606:4700:4700::1111" {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Error("could not find 1.1.1.1 in linked item queries")
		}
		if !foundV6 {
			t.Error("could not find 2606:4700:4700::1111 in linked item queries")
		}

		discovery.TestValidateItems(t, items)
	})

	t.Run("with an IP and therefore reverse DNS", func(t *testing.T) {
		s.ReverseLookup = true
		items, err := s.Search(context.Background(), "global", "1.1.1.1", false)

		if err != nil {
			t.Error(err)
		}

		// Make sure 1.1.1.1 is in there
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == "1.1.1.1" {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == "2606:4700:4700::1111" {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Error("could not find 1.1.1.1 in linked item queries")
		}
		if !foundV6 {
			t.Error("could not find 2606:4700:4700::1111 in linked item queries")
		}

		discovery.TestValidateItems(t, items)
	})
}

func TestDnsGet(t *testing.T) {
	t.Parallel()

	var conn net.Conn
	var err error

	// Check that we actually have an internet connection, if not there is not
	// point running this test
	conn, err = net.Dial("tcp", "one.one.one.one:443")
	conn.Close()

	if err != nil {
		t.Skip("No internet connection detected")
	}

	src := DNSAdapter{}

	t.Run("working request", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", "one.one.one.one", false)

		if err != nil {
			t.Fatal(err)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("bad dns entry", func(t *testing.T) {
		_, err := src.Get(context.Background(), "global", "something.does.not.exist.please.testing", false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		var e *sdp.QueryError
		if !errors.As(err, &e) {
			t.Errorf("expected error type to be *sdp.QueryError, got %T", err)
		}
	})

	t.Run("bad scope", func(t *testing.T) {
		_, err := src.Get(context.Background(), "something.local.test", "something.does.not.exist.please.testing", false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		var e *sdp.QueryError
		if !errors.As(err, &e) {
			t.Errorf("expected error type to be *sdp.QueryError, got %T", err)
		}
	})

	t.Run("with a CNAME", func(t *testing.T) {
		// When we do a Get on a CNAME, I wan it to work, but only return the
		// first thing
		item, err := src.Get(context.Background(), "global", "www.github.com", false)

		if err != nil {
			t.Fatal(err)
		}

		target := item.GetAttributes().GetAttrStruct().GetFields()["target"].GetStringValue()
		if target != "github.com" {
			t.Errorf("expected target to be github.com, got %v", target)
		}

		t.Log(item)
	})
}
