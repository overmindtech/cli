package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestLocationAdapter A adapter of `location` items for automated tests.
type TestLocationAdapter struct{}

// Type is the type of items that this returns
func (s *TestLocationAdapter) Type() string {
	return "test-location"
}

// Name Returns the name of the backend
func (s *TestLocationAdapter) Name() string {
	return "stdlib-test-location"
}

// Weighting of duplicate adapters
func (s *TestLocationAdapter) Weight() int {
	return 100
}

func (s *TestLocationAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

// List of scopes that this adapter is capable of find items for
func (s *TestLocationAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestLocationAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestLocationAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-london":
		return london(), nil
	case "test-soho":
		return soho(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestLocationAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{london(), soho()}, nil
}

func (d *TestLocationAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*":
		return []*sdp.Item{london(), soho()}, nil
	case "test-london":
		return []*sdp.Item{london()}, nil
	case "test-soho":
		return []*sdp.Item{soho()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
