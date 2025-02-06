package adapters

import (
	"context"
	"regexp"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func TestIPGet(t *testing.T) {
	src := IPAdapter{}

	t.Run("with ipv4 address", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", "213.21.3.187", false)

		if err != nil {
			t.Fatal(err)
		}

		if private, err := item.GetAttributes().Get("private"); err == nil {
			if private != false {
				t.Error("Expected itemAttributes.private to be false")
			}
		} else {
			t.Error("could not find 'private' attribute")
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("with ipv6 address", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", "2a01:4b00:8602:b600:5523:ce8d:dafc:3243", false)

		if err != nil {
			t.Fatal(err)
		}

		if private, err := item.GetAttributes().Get("private"); err == nil {
			if private != false {
				t.Error("Expected itemAttributes.private to be false")
			}
		} else {
			t.Error("could not find 'private' attribute")
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("with invalid address", func(t *testing.T) {
		_, err := src.Get(context.Background(), "global", "this is not valid", false)

		if err == nil {
			t.Error("expected error")
		} else {
			if matched, _ := regexp.MatchString("this is not valid", err.Error()); !matched {
				t.Errorf("expected error to contain 'this is not valid', got: %v", err)
			}
		}
	})

	t.Run("with ipv4 link-local address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			// Link-local addresses are not guaranteed to be unique beyond their
			// network segment, therefore routers do not forward packets with
			// link-local adapter or destination addresses. This means that it
			// doesn't make sense to have a "global" link-local address as it's
			// not truly global
			_, err := src.Get(context.Background(), "global", "169.254.1.25", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})

		t.Run("in another scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "some.computer", "169.254.1.25", false)

			if err != nil {
				t.Fatal(err)
			}

			if item.GetScope() != "some.computer" {
				t.Errorf("expected scope to be some.computer, got %v", item.GetScope())
			}

			if llu, err := item.GetAttributes().Get("linkLocalUnicast"); err != nil || llu == false {
				t.Errorf("expected linkLocalUnicast to be false, got %v", llu)
			}

			discovery.TestValidateItem(t, item)
		})
	})

	t.Run("with ipv4 private address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "global", "10.0.4.5", false)

			if err != nil {
				t.Fatal(err)
			}

			if p, err := item.GetAttributes().Get("private"); err != nil || p == false {
				t.Errorf("expected p to be true, got %v", p)
			}

			discovery.TestValidateItem(t, item)
		})

		t.Run("in another scope", func(t *testing.T) {
			_, err := src.Get(context.Background(), "some.computer", "10.0.4.5", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	})

	t.Run("with ipv4 loopback address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			// Link-local addresses are not guaranteed to be unique beyond their
			// network segment, therefore routers do not forward packets with
			// link-local adapter or destination addresses. This means that it
			// doesn't make sense to have a "global" link-local address as it's
			// not truly global
			_, err := src.Get(context.Background(), "global", "127.0.0.1", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})

		t.Run("in another scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "some.computer", "127.0.0.1", false)

			if err != nil {
				t.Fatal(err)
			}

			if item.GetScope() != "some.computer" {
				t.Errorf("expected scope to be some.computer, got %v", item.GetScope())
			}

			if loopback, err := item.GetAttributes().Get("loopback"); err != nil || loopback == false {
				t.Errorf("expected loopback to be false, got %v", loopback)
			}

			discovery.TestValidateItem(t, item)
		})
	})

	t.Run("with ipv6 link-local address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			// Link-local addresses are not guaranteed to be unique beyond their
			// network segment, therefore routers do not forward packets with
			// link-local adapter or destination addresses. This means that it
			// doesn't make sense to have a "global" link-local address as it's
			// not truly global
			_, err := src.Get(context.Background(), "global", "fe80::a70f:3a:338b:4801", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})

		t.Run("in another scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "some.computer", "fe80::a70f:3a:338b:4801", false)

			if err != nil {
				t.Fatal(err)
			}

			if item.GetScope() != "some.computer" {
				t.Errorf("expected scope to be some.computer, got %v", item.GetScope())
			}

			if llu, err := item.GetAttributes().Get("linkLocalUnicast"); err != nil || llu == false {
				t.Errorf("expected linkLocalUnicast top be false, got %v", llu)
			}

			discovery.TestValidateItem(t, item)
		})
	})

	t.Run("with ipv6 private address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "global", "fd12:3456:789a:1::1", false)

			if err != nil {
				t.Fatal(err)
			}

			if p, err := item.GetAttributes().Get("private"); err != nil || p == false {
				t.Errorf("expected p to be true, got %v", p)
			}

			discovery.TestValidateItem(t, item)

		})

		t.Run("in another scope", func(t *testing.T) {
			_, err := src.Get(context.Background(), "some.computer", "fd12:3456:789a:1::1", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	})

	t.Run("with ipv6 loopback address", func(t *testing.T) {
		t.Run("in the global scope", func(t *testing.T) {
			// Link-local addresses are not guaranteed to be unique beyond their
			// network segment, therefore routers do not forward packets with
			// link-local adapter or destination addresses. This means that it
			// doesn't make sense to have a "global" link-local address as it's
			// not truly global
			_, err := src.Get(context.Background(), "global", "::1", false)

			if err == nil {
				t.Error("expected error but got nil")
			}
		})

		t.Run("in another scope", func(t *testing.T) {
			item, err := src.Get(context.Background(), "some.computer", "::1", false)

			if err != nil {
				t.Fatal(err)
			}

			if item.GetScope() != "some.computer" {
				t.Errorf("expected scope to be some.computer, got %v", item.GetScope())
			}

			if loopback, err := item.GetAttributes().Get("loopback"); err != nil || loopback == false {
				t.Errorf("expected loopback to be false, got %v", loopback)
			}

			discovery.TestValidateItem(t, item)
		})
	})

	t.Run("with a wildcard scope", func(t *testing.T) {
		item, err := src.Get(context.Background(), sdp.WILDCARD, "213.21.3.187", false)

		if err != nil {
			t.Fatal(err)
		}

		if private, err := item.GetAttributes().Get("private"); err == nil {
			if private != false {
				t.Error("Expected itemAttributes.private to be false")
			}
		} else {
			t.Error("could not find 'private' attribute")
		}

		discovery.TestValidateItem(t, item)
	})
}
