package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestGroupAdapter A adapter of `group` items for automated tests.
type TestGroupAdapter struct{}

// Type is the type of items that this returns
func (s *TestGroupAdapter) Type() string {
	return "test-group"
}

// Name Returns the name of the backend
func (s *TestGroupAdapter) Name() string {
	return "stdlib-test-group"
}

// Weighting of duplicate adapters
func (s *TestGroupAdapter) Weight() int {
	return 100
}

func (s *TestGroupAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

// List of scopes that this adapter is capable of find items for
func (s *TestGroupAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestGroupAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestGroupAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-admins":
		return admins(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestGroupAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{admins()}, nil
}

func (d *TestGroupAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-admins":
		return []*sdp.Item{admins()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
