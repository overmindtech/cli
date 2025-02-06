package test

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
)

// TestHobbyAdapter A adapter of `hobby` items for automated tests.
type TestHobbyAdapter struct{}

// Type is the type of items that this returns
func (s *TestHobbyAdapter) Type() string {
	return "test-hobby"
}

// Name Returns the name of the backend
func (s *TestHobbyAdapter) Name() string {
	return "stdlib-test-hobby"
}

func (s *TestHobbyAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: s.Name(),
	}
}

// Weighting of duplicate adapters
func (s *TestHobbyAdapter) Weight() int {
	return 100
}

// List of scopes that this adapter is capable of find items for
func (s *TestHobbyAdapter) Scopes() []string {
	return []string{
		"test",
	}
}

func (s *TestHobbyAdapter) Hidden() bool {
	return true
}

// Gets a single item. This expects a name
func (d *TestHobbyAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "test-motorcycling":
		return motorcycling(), nil
	case "test-knitting":
		return knitting(), nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}

func (d *TestHobbyAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	return []*sdp.Item{motorcycling(), knitting()}, nil
}

func (d *TestHobbyAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "test" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "test queries only supported in 'test' scope",
			Scope:       scope,
		}
	}

	switch query {
	case "", "*", "test-motorcycling":
		return []*sdp.Item{motorcycling()}, nil
	default:
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     scope,
		}
	}
}
