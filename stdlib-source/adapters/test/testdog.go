package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestDogAdapter An adapter of `dog` items for automated tests.
type TestDogAdapter struct{}

// Type is the type of items that this returns
func (s *TestDogAdapter) Type() string {
	return "test-dog"
}

// Name Returns the name of the backend
func (s *TestDogAdapter) Name() string {
	return "stdlib-test-dog"
}

// Metadata Returns the metadata for the adapter
func (s *TestDogAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

// Weighting of duplicate adapters
func (s *TestDogAdapter) Weight() int {
	return 100
}

// List of scopes that this adapter is capable of find items for
func (s *TestDogAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestDogAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestDogAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-manny":
		return manny(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestDogAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{manny()}, nil
}

func (d *TestDogAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-manny":
		return []*sdp.Item{manny()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
