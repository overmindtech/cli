package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestRegionAdapter A adapter of `region` items for automated tests.
type TestRegionAdapter struct{}

// Type is the type of items that this returns
func (s *TestRegionAdapter) Type() string {
	return "test-region"
}

// Name Returns the name of the backend
func (s *TestRegionAdapter) Name() string {
	return "stdlib-test-region"
}

// Weighting of duplicate adapters
func (s *TestRegionAdapter) Weight() int {
	return 100
}

// List of scopes that this adapter is capable of find items for
func (s *TestRegionAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestRegionAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

func (s *TestRegionAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestRegionAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-gb":
		return gb(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestRegionAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{gb()}, nil
}

func (d *TestRegionAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-gb":
		return []*sdp.Item{gb()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
