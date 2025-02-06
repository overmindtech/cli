package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestPersonAdapter A adapter of `person` items for automated tests.
type TestPersonAdapter struct{}

// Type is the type of items that this returns
func (s *TestPersonAdapter) Type() string {
	return "test-person"
}

// Name Returns the name of the backend
func (s *TestPersonAdapter) Name() string {
	return "stdlib-test-person"
}

// Weighting of duplicate adapters
func (s *TestPersonAdapter) Weight() int {
	return 100
}

// List of scopes that this adapter is capable of find items for
func (s *TestPersonAdapter) Scopes() []string {
	return []string{
		"test",
	}
}
func (s *TestPersonAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

func (s *TestPersonAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestPersonAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-dylan":
		return dylan(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestPersonAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{dylan()}, nil
}

func (d *TestPersonAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-dylan":
		return []*sdp.Item{dylan()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
