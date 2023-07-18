package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestChangingItemQueriesFromPlan(t *testing.T) {
	testFile := "testdata/plan.json"
	planJSON, err := os.ReadFile(testFile)

	if err != nil {
		t.Errorf("Error reading %v: %v", testFile, err)
	}

	queries, err := changingItemQueriesFromPlan(context.Background(), planJSON, logrus.Fields{})

	if err != nil {
		t.Error(err)
	}

	if len(queries) != 1 {
		t.Errorf("Expected 1 queries, got %v", len(queries))
	}

	if queries[0].Type != "iam-policy" {
		t.Errorf("Expected query type to be iam-policy, got %v", queries[0].Type)
	}

	if queries[0].Query != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected query to be arn:aws:iam::123456789012:policy/test-alb-ingress, got %v", queries[0].Query)
	}

	if queries[0].Scope != "*" {
		t.Errorf("Expected query scope to be *, got %v", queries[0].Scope)
	}
}
