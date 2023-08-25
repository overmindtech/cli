package cmd

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestChangingItemQueriesFromPlan(t *testing.T) {
	queries, err := changingItemQueriesFromPlan(context.Background(), "testdata/plan.json", logrus.Fields{})

	if err != nil {
		t.Error(err)
	}

	if len(queries) != 3 {
		t.Errorf("Expected 3 queries, got %v", len(queries))
	}

	if queries[0].Type != "Deployment" {
		t.Errorf("Expected query type to be Deployment, got %v", queries[0].Type)
	}

	if queries[0].Query != "nats-box" {
		t.Errorf("Expected query to be nats-box, got %v", queries[0].Query)
	}

	// Since this resource is being deleted it doesn't have any config so we
	// can't determine the scope from mappings
	if queries[0].Scope != "*" {
		t.Errorf("Expected query scope to be *, got %v", queries[0].Scope)
	}

	if queries[1].Type != "Deployment" {
		t.Errorf("Expected query type to be Deployment, got %v", queries[1].Type)
	}

	if queries[1].Query != "api-server" {
		t.Errorf("Expected query to be api-server, got %v", queries[1].Query)
	}

	if queries[1].Scope != "dogfood.default" {
		t.Errorf("Expected query scope to be dogfood.default, got %v", queries[1].Scope)
	}

	if queries[2].Type != "iam-policy" {
		t.Errorf("Expected query type to be iam-policy, got %v", queries[2].Type)
	}

	if queries[2].Query != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected query to be arn:aws:iam::123456789012:policy/test-alb-ingress, got %v", queries[2].Query)
	}

	if queries[2].Scope != "*" {
		t.Errorf("Expected query scope to be *, got %v", queries[2].Scope)
	}
}
