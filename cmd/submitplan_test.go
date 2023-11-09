package cmd

import (
	"context"
	"testing"

	"github.com/overmindtech/sdp-go"
	"github.com/sirupsen/logrus"
)

func TestMappedItemDiffsFromPlan(t *testing.T) {
	mappedItemDiffs, err := mappedItemDiffsFromPlan(context.Background(), "testdata/plan.json", logrus.Fields{})

	if err != nil {
		t.Error(err)
	}

	if len(mappedItemDiffs) != 3 {
		t.Errorf("Expected 3 changes, got %v:", len(mappedItemDiffs))
		for _, diff := range mappedItemDiffs {
			t.Errorf("  %v", diff)
		}
	}

	var nats_box_deployment *sdp.MappedItemDiff
	var api_server_deployment *sdp.MappedItemDiff
	var aws_iam_policy *sdp.MappedItemDiff

	for _, diff := range mappedItemDiffs {
		item := diff.Item.Before
		if item == nil && diff.Item.After != nil {
			item = diff.Item.After
		}
		if item == nil {
			t.Errorf("Expected any of before/after items to be set, but there's nothing: %v", diff)
			continue
		}

		// t.Logf("item: %v", item.Attributes.AttrStruct.Fields["terraform_address"].GetStringValue())
		if item.Attributes.AttrStruct.Fields["terraform_address"].GetStringValue() == "kubernetes_deployment.nats_box" {
			nats_box_deployment = diff
		} else if item.Attributes.AttrStruct.Fields["terraform_address"].GetStringValue() == "kubernetes_deployment.api_server" {
			api_server_deployment = diff
		} else if item.Type == "iam-policy" {
			aws_iam_policy = diff
		}
	}

	// check nats_box_deployment
	t.Logf("nats_box_deployment: %v", nats_box_deployment)
	if nats_box_deployment == nil {
		t.Fatalf("Expected nats_box_deployment to be set, but it's not")
	}
	if nats_box_deployment.Item.Status != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED {
		t.Errorf("Expected nats_box_deployment status to be 'deleted', but it's '%v'", nats_box_deployment.Item.Status)
	}
	if nats_box_deployment.MappingQuery.Type != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.MappingQuery.Type)
	}
	if nats_box_deployment.MappingQuery.Query != "nats-box" {
		t.Errorf("Expected nats_box_deployment query to be 'nats-box', got '%v'", nats_box_deployment.MappingQuery.Query)
	}
	if nats_box_deployment.MappingQuery.Scope != "*" {
		t.Errorf("Expected nats_box_deployment query scope to be '*', got '%v'", nats_box_deployment.MappingQuery.Scope)
	}
	if nats_box_deployment.Item.Before.Scope != "terraform_plan" {
		t.Errorf("Expected nats_box_deployment before item scope to be 'terraform_plan', got '%v'", nats_box_deployment.Item.Before.Scope)
	}
	if nats_box_deployment.MappingQuery.Type != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.MappingQuery.Type)
	}
	if nats_box_deployment.Item.Before.Type != "Deployment" {
		t.Errorf("Expected nats_box_deployment before item type to be 'Deployment', got '%v'", nats_box_deployment.Item.Before.Type)
	}
	if nats_box_deployment.MappingQuery.Query != "nats-box" {
		t.Errorf("Expected nats_box_deployment query query to be 'nats-box', got '%v'", nats_box_deployment.MappingQuery.Query)
	}

	// check api_server_deployment
	t.Logf("api_server_deployment: %v", api_server_deployment)
	if api_server_deployment == nil {
		t.Fatalf("Expected api_server_deployment to be set, but it's not")
	}
	if api_server_deployment.Item.Status != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected api_server_deployment status to be 'updated', but it's '%v'", api_server_deployment.Item.Status)
	}
	if api_server_deployment.MappingQuery.Type != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.MappingQuery.Type)
	}
	if api_server_deployment.MappingQuery.Query != "api-server" {
		t.Errorf("Expected api_server_deployment query to be 'api-server', got '%v'", api_server_deployment.MappingQuery.Query)
	}
	if api_server_deployment.MappingQuery.Scope != "dogfood.default" {
		t.Errorf("Expected api_server_deployment query scope to be 'dogfood.default', got '%v'", api_server_deployment.MappingQuery.Scope)
	}
	if api_server_deployment.Item.Before.Scope != "dogfood.default" {
		t.Errorf("Expected api_server_deployment before item scope to be 'dogfood.default', got '%v'", api_server_deployment.Item.Before.Scope)
	}
	if api_server_deployment.MappingQuery.Type != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.MappingQuery.Type)
	}
	if api_server_deployment.Item.Before.Type != "Deployment" {
		t.Errorf("Expected api_server_deployment before item type to be 'Deployment', got '%v'", api_server_deployment.Item.Before.Type)
	}
	if api_server_deployment.MappingQuery.Query != "api-server" {
		t.Errorf("Expected api_server_deployment query query to be 'api-server', got '%v'", api_server_deployment.MappingQuery.Query)
	}

	// check aws_iam_policy
	t.Logf("aws_iam_policy: %v", aws_iam_policy)
	if aws_iam_policy == nil {
		t.Fatalf("Expected aws_iam_policy to be set, but it's not")
	}
	if aws_iam_policy.Item.Status != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected aws_iam_policy status to be 'updated', but it's %v", aws_iam_policy.Item.Status)
	}
	if aws_iam_policy.MappingQuery.Type != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.MappingQuery.Type)
	}
	if aws_iam_policy.MappingQuery.Query != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.MappingQuery.Query)
	}
	if aws_iam_policy.MappingQuery.Scope != "*" {
		t.Errorf("Expected aws_iam_policy query scope to be '*', got '%v'", aws_iam_policy.MappingQuery.Scope)
	}
	if aws_iam_policy.Item.Before.Scope != "terraform_plan" {
		t.Errorf("Expected aws_iam_policy before item scope to be 'terraform_plan', got '%v'", aws_iam_policy.Item.Before.Scope)
	}
	if aws_iam_policy.MappingQuery.Type != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.MappingQuery.Type)
	}
	if aws_iam_policy.Item.Before.Type != "iam-policy" {
		t.Errorf("Expected aws_iam_policy before item type to be 'iam-policy', got '%v'", aws_iam_policy.Item.Before.Type)
	}
	if aws_iam_policy.MappingQuery.Query != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.MappingQuery.Query)
	}
}
