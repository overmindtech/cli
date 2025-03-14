package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestFoodAdapter A adapter of `food` items for automated tests.
type TestFoodAdapter struct{}

// Type is the type of items that this returns
func (s *TestFoodAdapter) Type() string {
	return "test-food"
}

// Name Returns the name of the backend
func (s *TestFoodAdapter) Name() string {
	return "stdlib-test-food"
}

func (s *TestFoodAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

// Weighting of duplicate adapters
func (s *TestFoodAdapter) Weight() int {
	return 100
}

// List of scopes that this adapter is capable of find items for
func (s *TestFoodAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestFoodAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestFoodAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-kibble":
		return kibble(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestFoodAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{kibble()}, nil
}

func (d *TestFoodAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-kibble":
		return []*sdp.Item{kibble()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
