package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/api/pubsub/v1"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestPubSubSubscription(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	subscriptionName := "test-subscription"

	subscription := &pubsub.Subscription{
		Name:  fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionName),
		Topic: fmt.Sprintf("projects/%s/topics/test-topic", projectID),
		DeadLetterPolicy: &pubsub.DeadLetterPolicy{
			DeadLetterTopic:     fmt.Sprintf("projects/%s/topics/dead-letter-topic", projectID),
			MaxDeliveryAttempts: 5,
		},
		PushConfig: &pubsub.PushConfig{
			PushEndpoint: "https://example.com/push-endpoint",
			OidcToken: &pubsub.OidcToken{
				ServiceAccountEmail: fmt.Sprintf("push-sa@%s.iam.gserviceaccount.com", projectID),
				Audience:            "https://example.com",
			},
		},
		BigqueryConfig: &pubsub.BigQueryConfig{
			Table:               "test-project.test_dataset.test_table",
			ServiceAccountEmail: fmt.Sprintf("bq-sa@%s.iam.gserviceaccount.com", projectID),
		},
		CloudStorageConfig: &pubsub.CloudStorageConfig{
			Bucket:              "test-bucket",
			ServiceAccountEmail: fmt.Sprintf("storage-sa@%s.iam.gserviceaccount.com", projectID),
		},
	}

	subscriptionList := &pubsub.ListSubscriptionsResponse{
		Subscriptions: []*pubsub.Subscription{subscription},
	}

	sdpItemType := gcpshared.PubSubSubscription

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s", projectID, subscriptionName): {
			StatusCode: http.StatusOK,
			Body:       subscription,
		},
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions", projectID): {
			StatusCode: http.StatusOK,
			Body:       subscriptionList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, subscriptionName, true)
		if err != nil {
			t.Fatalf("Failed to get subscription: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// topic
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-topic",
					ExpectedScope:  projectID,
				},
				{
					// deadLetterPolicy.deadLetterTopic
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "dead-letter-topic",
					ExpectedScope:  projectID,
				},
				{
					// pushConfig.pushEndpoint
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://example.com/push-endpoint",
					ExpectedScope:  "global",
				},
				{
					// pushConfig.oidcToken.serviceAccountEmail
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("push-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
				},
				{
					// bigqueryConfig.table
					ExpectedType:   gcpshared.BigQueryTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test_dataset", "test_table"),
					ExpectedScope:  projectID,
				},
				{
					// bigqueryConfig.serviceAccountEmail
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("bq-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
				},
				{
					// cloudStorageConfig.bucket
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-bucket",
					ExpectedScope:  projectID,
				},
				{
					// cloudStorageConfig.serviceAccountEmail
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("storage-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list subscriptions: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 subscription, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s", projectID, subscriptionName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]any{"error": "Subscription not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, subscriptionName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent subscription, but got nil")
		}
	})
}

// TestPubSubSubscriptionIAMTerraformMappings verifies that the IAM Terraform resource
// types (iam_binding, iam_member, iam_policy) are registered as terraform mappings on
// the PubSub Subscription adapter. This is critical because these Terraform-only
// resources don't have their own GCP API — they represent IAM policy changes on the
// parent subscription. Without these mappings, IAM changes would show as "Unsupported"
// in the change analysis UI instead of being resolved to the parent subscription for
// blast radius analysis.
//
// Background: google_pubsub_subscription_iam_binding is an authoritative Terraform
// resource that manages a single role's members on a subscription. When it changes,
// we need to resolve it to the affected subscription so customers see the downstream
// impact (e.g. services that read from the subscription losing access).
func TestPubSubSubscriptionIAMTerraformMappings(t *testing.T) {
	// Retrieve the terraform mappings registered for PubSubSubscription
	tfMapping, ok := gcpshared.SDPAssetTypeToTerraformMappings[gcpshared.PubSubSubscription]
	if !ok {
		t.Fatal("Expected PubSubSubscription to have terraform mappings registered, but none were found")
	}

	// Build a lookup of terraform type -> query field from the registered mappings.
	// This mirrors the logic in cli/tfutils/plan_mapper.go that splits
	// TerraformQueryMap on "." to get the terraform type and attribute name.
	type mappingInfo struct {
		terraformType string
		queryField    string
		method        sdp.QueryMethod
	}
	registeredMappings := make([]mappingInfo, 0, len(tfMapping.Mappings))
	for _, m := range tfMapping.Mappings {
		parts := strings.SplitN(m.GetTerraformQueryMap(), ".", 2)
		if len(parts) != 2 {
			t.Errorf("Invalid TerraformQueryMap format: %q (expected 'type.attribute')", m.GetTerraformQueryMap())
			continue
		}
		registeredMappings = append(registeredMappings, mappingInfo{
			terraformType: parts[0],
			queryField:    parts[1],
			method:        m.GetTerraformMethod(),
		})
	}

	// Define the IAM terraform types we expect to be mapped, along with the
	// Terraform attribute that identifies the parent subscription.
	// All three IAM resource types use "subscription" as the attribute that
	// contains the subscription name.
	expectedIAMMappings := []struct {
		terraformType string
		queryField    string
		method        sdp.QueryMethod
		description   string // documents why this mapping exists, for reviewer clarity
	}{
		{
			terraformType: "google_pubsub_subscription_iam_binding",
			queryField:    "subscription",
			method:        sdp.QueryMethod_GET,
			description:   "Authoritative for a given role — maps to parent subscription for blast radius",
		},
		{
			terraformType: "google_pubsub_subscription_iam_member",
			queryField:    "subscription",
			method:        sdp.QueryMethod_GET,
			description:   "Non-authoritative single member — maps to parent subscription for blast radius",
		},
		{
			terraformType: "google_pubsub_subscription_iam_policy",
			queryField:    "subscription",
			method:        sdp.QueryMethod_GET,
			description:   "Authoritative for full IAM policy — maps to parent subscription for blast radius",
		},
	}

	for _, expected := range expectedIAMMappings {
		t.Run(expected.terraformType, func(t *testing.T) {
			found := false
			for _, registered := range registeredMappings {
				if registered.terraformType == expected.terraformType {
					found = true

					if registered.queryField != expected.queryField {
						t.Errorf("Terraform type %s: expected query field %q, got %q",
							expected.terraformType, expected.queryField, registered.queryField)
					}

					if registered.method != expected.method {
						t.Errorf("Terraform type %s: expected method %s, got %s",
							expected.terraformType, expected.method, registered.method)
					}
					break
				}
			}

			if !found {
				t.Errorf("Terraform type %s is not registered as a mapping on PubSubSubscription. "+
					"This means %q changes will show as 'Unsupported' in the change analysis UI. "+
					"Purpose: %s",
					expected.terraformType, expected.terraformType, expected.description)
			}
		})
	}

	// Also verify the base subscription mapping still exists (sanity check)
	t.Run("google_pubsub_subscription", func(t *testing.T) {
		found := false
		for _, registered := range registeredMappings {
			if registered.terraformType == "google_pubsub_subscription" {
				found = true
				if registered.queryField != "name" {
					t.Errorf("Expected query field 'name' for google_pubsub_subscription, got %q", registered.queryField)
				}
				break
			}
		}
		if !found {
			t.Error("Base terraform mapping for google_pubsub_subscription is missing — this would break all subscription change analysis")
		}
	})
}
