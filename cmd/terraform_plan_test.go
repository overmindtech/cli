package cmd

import (
	"context"
	"testing"

	"github.com/overmindtech/sdp-go"
	"github.com/sirupsen/logrus"
)

func TestMappedItemDiffsFromPlan(t *testing.T) {
	mappedItemDiffs, err := mappedItemDiffsFromPlanFile(context.Background(), "testdata/plan.json", logrus.Fields{})

	if err != nil {
		t.Error(err)
	}

	if len(mappedItemDiffs) != 5 {
		t.Errorf("Expected 5 changes, got %v:", len(mappedItemDiffs))
		for _, diff := range mappedItemDiffs {
			t.Errorf("  %v", diff)
		}
	}

	var nats_box_deployment *sdp.MappedItemDiff
	var api_server_deployment *sdp.MappedItemDiff
	var aws_iam_policy *sdp.MappedItemDiff
	var secret *sdp.MappedItemDiff

	for _, diff := range mappedItemDiffs {
		item := diff.GetItem().GetBefore()
		if item == nil && diff.GetItem().GetAfter() != nil {
			item = diff.GetItem().GetAfter()
		}
		if item == nil {
			t.Errorf("Expected any of before/after items to be set, but there's nothing: %v", diff)
			continue
		}

		// t.Logf("item: %v", item.Attributes.AttrStruct.Fields["terraform_address"].GetStringValue())
		if item.GetAttributes().GetAttrStruct().GetFields()["terraform_address"].GetStringValue() == "kubernetes_deployment.nats_box" {
			if nats_box_deployment != nil {
				t.Errorf("Found multiple nats_box_deployment: %v, %v", nats_box_deployment, diff)
			}
			nats_box_deployment = diff
		} else if item.GetAttributes().GetAttrStruct().GetFields()["terraform_address"].GetStringValue() == "kubernetes_deployment.api_server" {
			if api_server_deployment != nil {
				t.Errorf("Found multiple api_server_deployment: %v, %v", api_server_deployment, diff)
			}
			api_server_deployment = diff
		} else if item.GetType() == "iam-policy" {
			if aws_iam_policy != nil {
				t.Errorf("Found multiple aws_iam_policy: %v, %v", aws_iam_policy, diff)
			}
			aws_iam_policy = diff
		} else if item.GetType() == "Secret" {
			if secret != nil {
				t.Errorf("Found multiple secrets: %v, %v", secret, diff)
			}
			secret = diff
		}
	}

	// check nats_box_deployment
	t.Logf("nats_box_deployment: %v", nats_box_deployment)
	if nats_box_deployment == nil {
		t.Fatalf("Expected nats_box_deployment to be set, but it's not")
	}
	if nats_box_deployment.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED {
		t.Errorf("Expected nats_box_deployment status to be 'deleted', but it's '%v'", nats_box_deployment.GetItem().GetStatus())
	}
	if nats_box_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.GetMappingQuery().GetType())
	}
	if nats_box_deployment.GetMappingQuery().GetQuery() != "nats-box" {
		t.Errorf("Expected nats_box_deployment query to be 'nats-box', got '%v'", nats_box_deployment.GetMappingQuery().GetQuery())
	}
	if nats_box_deployment.GetMappingQuery().GetScope() != "*" {
		t.Errorf("Expected nats_box_deployment query scope to be '*', got '%v'", nats_box_deployment.GetMappingQuery().GetScope())
	}
	if nats_box_deployment.GetItem().GetBefore().GetScope() != "terraform_plan" {
		t.Errorf("Expected nats_box_deployment before item scope to be 'terraform_plan', got '%v'", nats_box_deployment.GetItem().GetBefore().GetScope())
	}
	if nats_box_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.GetMappingQuery().GetType())
	}
	if nats_box_deployment.GetItem().GetBefore().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment before item type to be 'Deployment', got '%v'", nats_box_deployment.GetItem().GetBefore().GetType())
	}
	if nats_box_deployment.GetMappingQuery().GetQuery() != "nats-box" {
		t.Errorf("Expected nats_box_deployment query query to be 'nats-box', got '%v'", nats_box_deployment.GetMappingQuery().GetQuery())
	}

	// check api_server_deployment
	t.Logf("api_server_deployment: %v", api_server_deployment)
	if api_server_deployment == nil {
		t.Fatalf("Expected api_server_deployment to be set, but it's not")
	}
	if api_server_deployment.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected api_server_deployment status to be 'updated', but it's '%v'", api_server_deployment.GetItem().GetStatus())
	}
	if api_server_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.GetMappingQuery().GetType())
	}
	if api_server_deployment.GetMappingQuery().GetQuery() != "api-server" {
		t.Errorf("Expected api_server_deployment query to be 'api-server', got '%v'", api_server_deployment.GetMappingQuery().GetQuery())
	}
	if api_server_deployment.GetMappingQuery().GetScope() != "dogfood.default" {
		t.Errorf("Expected api_server_deployment query scope to be 'dogfood.default', got '%v'", api_server_deployment.GetMappingQuery().GetScope())
	}
	if api_server_deployment.GetItem().GetBefore().GetScope() != "dogfood.default" {
		t.Errorf("Expected api_server_deployment before item scope to be 'dogfood.default', got '%v'", api_server_deployment.GetItem().GetBefore().GetScope())
	}
	if api_server_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.GetMappingQuery().GetType())
	}
	if api_server_deployment.GetItem().GetBefore().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment before item type to be 'Deployment', got '%v'", api_server_deployment.GetItem().GetBefore().GetType())
	}
	if api_server_deployment.GetMappingQuery().GetQuery() != "api-server" {
		t.Errorf("Expected api_server_deployment query query to be 'api-server', got '%v'", api_server_deployment.GetMappingQuery().GetQuery())
	}

	// check aws_iam_policy
	t.Logf("aws_iam_policy: %v", aws_iam_policy)
	if aws_iam_policy == nil {
		t.Fatalf("Expected aws_iam_policy to be set, but it's not")
	}
	if aws_iam_policy.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected aws_iam_policy status to be 'updated', but it's %v", aws_iam_policy.GetItem().GetStatus())
	}
	if aws_iam_policy.GetMappingQuery().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.GetMappingQuery().GetType())
	}
	if aws_iam_policy.GetMappingQuery().GetQuery() != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.GetMappingQuery().GetQuery())
	}
	if aws_iam_policy.GetMappingQuery().GetScope() != "*" {
		t.Errorf("Expected aws_iam_policy query scope to be '*', got '%v'", aws_iam_policy.GetMappingQuery().GetScope())
	}
	if aws_iam_policy.GetItem().GetBefore().GetScope() != "terraform_plan" {
		t.Errorf("Expected aws_iam_policy before item scope to be 'terraform_plan', got '%v'", aws_iam_policy.GetItem().GetBefore().GetScope())
	}
	if aws_iam_policy.GetMappingQuery().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.GetMappingQuery().GetType())
	}
	if aws_iam_policy.GetItem().GetBefore().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy before item type to be 'iam-policy', got '%v'", aws_iam_policy.GetItem().GetBefore().GetType())
	}
	if aws_iam_policy.GetMappingQuery().GetQuery() != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.GetMappingQuery().GetQuery())
	}

	// check secret
	t.Logf("secret: %v", secret)
	if secret == nil {
		t.Fatalf("Expected secret to be set, but it's not")
	}
	if secret.GetMappingQuery().GetScope() != "dogfood.default" {
		t.Errorf("Expected secret query scope to be 'dogfood.default', got '%v'", secret.GetMappingQuery().GetScope())
	}
}