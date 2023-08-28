package cmd

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestChangingItemQueriesFromPlan(t *testing.T) {
	mappedPlan, err := changingItemQueriesFromPlan(context.Background(), "testdata/plan.json", logrus.Fields{})

	if err != nil {
		t.Error(err)
	}

	deployments, exists := mappedPlan.SupportedChanges["kubernetes_deployment"]

	if !exists {
		t.Errorf("Expected kubernetes_deployment to be in supported changes")
	}

	if len(deployments) != 2 {
		t.Errorf("Expected 2 deployments, got %v", len(deployments))
	}

	if deployments[0].OvermindQuery.Type != "Deployment" {
		t.Errorf("Expected query type to be Deployment, got %v", deployments[0].OvermindQuery.Type)
	}

	if deployments[0].OvermindQuery.Query != "nats-box" {
		t.Errorf("Expected query to be nats-box, got %v", deployments[0].OvermindQuery.Query)
	}

	if deployments[0].OvermindQuery.Scope != "*" {
		t.Errorf("Expected query scope to be *, got %v", deployments[0].OvermindQuery.Scope)
	}

	if deployments[1].OvermindQuery.Type != "Deployment" {
		t.Errorf("Expected query type to be Deployment, got %v", deployments[1].OvermindQuery.Type)
	}

	if deployments[1].OvermindQuery.Query != "api-server" {
		t.Errorf("Expected query to be api-server, got %v", deployments[1].OvermindQuery.Query)
	}

	if deployments[1].OvermindQuery.Scope != "dogfood.default" {
		t.Errorf("Expected query scope to be dogfood.default, got %v", deployments[1].OvermindQuery.Scope)
	}

	iamPolicies, exists := mappedPlan.SupportedChanges["aws_iam_policy"]

	if !exists {
		t.Errorf("Expected aws_iam_policy to be in supported changes")
	}

	if len(iamPolicies) != 1 {
		t.Errorf("Expected 1 iam policy, got %v", len(iamPolicies))
	}

	if iamPolicies[0].OvermindQuery.Type != "iam-policy" {
		t.Errorf("Expected query type to be iam-policy, got %v", iamPolicies[0].OvermindQuery.Type)
	}

	if iamPolicies[0].OvermindQuery.Query != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected query to be arn:aws:iam::123456789012:policy/test-alb-ingress, got %v", iamPolicies[0].OvermindQuery.Query)
	}

	if iamPolicies[0].OvermindQuery.Scope != "*" {
		t.Errorf("Expected query scope to be *, got %v", iamPolicies[0].OvermindQuery.Scope)
	}
}
