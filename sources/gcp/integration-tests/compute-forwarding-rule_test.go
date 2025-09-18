package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2/apierror"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeForwardingRuleIntegration(t *testing.T) {
	// TODO: Implement the dependencies for Compute Forwarding Rule
	// This test currently asserts that the GCP SDK client satisfies the adapter interface
	t.Skipf("Skipping integration test for Compute Forwarding Rule until we implement the dependencies: BackendService, or Load Balancer and Target HTTP Proxy")
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	region := os.Getenv("GCP_REGION")
	if region == "" {
		t.Skip("GCP_REGION environment variable not set")
	}

	ruleName := "integration-test-forwarding-rule"

	ctx := context.Background()

	// Create a new Compute Forwarding Rule client
	client, err := compute.NewForwardingRulesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewForwardingRulesRESTClient: %v", err)
	}
	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeForwardingRule(ctx, client, projectID, region, ruleName)
		if err != nil {
			t.Fatalf("Failed to create forwarding rule: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Forwarding Rule in project %s, region %s", projectID, region)

		ruleWrapper := manual.NewComputeForwardingRule(gcpshared.NewComputeForwardingRuleClient(client), projectID, region)
		scope := ruleWrapper.Scopes()[0]

		ruleAdapter := sources.WrapperToAdapter(ruleWrapper)
		sdpItem, qErr := ruleAdapter.Get(ctx, scope, ruleName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem == nil {
			t.Fatalf("Expected sdpItem to be non-nil")
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()
		uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if err != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != ruleName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", ruleName, uniqueAttrValue)
		}

		// Check if adapter supports listing
		listable, ok := ruleAdapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list forwarding rules: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one forwarding rule, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == ruleName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find forwarding rule %s in the list", ruleName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteComputeForwardingRule(ctx, client, projectID, region, ruleName)
		if err != nil {
			t.Fatalf("Failed to delete forwarding rule: %v", err)
		}
	})
}

// createComputeForwardingRule creates a GCP Compute Forwarding Rule with the given parameters.
func createComputeForwardingRule(ctx context.Context, client *compute.ForwardingRulesClient, projectID, region, ruleName string) error {
	req := &computepb.InsertForwardingRuleRequest{
		Project: projectID,
		Region:  region,
		ForwardingRuleResource: &computepb.ForwardingRule{
			Name: ptr.To(ruleName),
			// IP address for which this forwarding rule accepts traffic.
			// When a client sends traffic to this IP address, the forwarding rule directs the traffic to the referenced target or backendService.
			// While creating a forwarding rule, specifying an IPAddress is required under the following circumstances:
			//	- When the target is set to targetGrpcProxy and validateForProxyless is set to true, the IPAddress should be set to 0.0.0.0.
			//	- When the target is a Private Service Connect Google APIs bundle, you must specify an IPAddress.
			//	Otherwise, you can optionally specify an IP address that references an existing static (reserved) IP address resource.
			//	When omitted, Google Cloud assigns an ephemeral IP address.
			//	Use one of the following formats to specify an IP address while creating a forwarding rule:
			//	* IP address number, as in `100.1.2.3`
			//	* IPv6 address range, as in `2600:1234::/96`
			//	* Full resource URL, as in https://www.googleapis.com/compute/v1/projects/ project_id/regions/region/addresses/address-name
			//	* Partial URL or by name, as in:
			//		- projects/project_id/regions/region/addresses/address-name
			//		- regions/region/addresses/address-name
			//		- global/addresses/address-name
			//		- address-name
			//	The forwarding rule's target or backendService, and in most cases, also the loadBalancingScheme,
			//	determine the type of IP address that you can use.
			//	For detailed information, see [IP address specifications](https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts#ip_address_specifications).
			//	When reading an IPAddress, the API always returns the IP address number.
			IPAddress:  ptr.To("192.168.1.1"),
			IPProtocol: ptr.To("TCP"),
			PortRange:  ptr.To("80-80"),
			// The URL of the target resource to receive the matched traffic.
			// For regional forwarding rules, this target must be in the same region as the forwarding rule.
			// For global forwarding rules, this target must be a global load balancing resource.
			// The forwarded traffic must be of a type appropriate to the target object.
			//- For load balancers, see the "Target" column in [Port specifications](https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts#ip_address_specifications).
			//- For Private Service Connect forwarding rules that forward traffic to Google APIs, provide the name of a supported Google API bundle:
			//- vpc-sc - APIs that support VPC Service Controls.
			//- all-apis - All supported Google APIs.
			//- For Private Service Connect forwarding rules that forward traffic to managed services, the target must be a service attachment.
			//The target is not mutable once set as a service attachment.
			Target: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-target-pool"),
		},
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}

	log.Printf("Forwarding rule %s created successfully in project %s, region %s", ruleName, projectID, region)
	return nil
}

func deleteComputeForwardingRule(ctx context.Context, client *compute.ForwardingRulesClient, projectID, region, ruleName string) error {
	req := &computepb.DeleteForwardingRuleRequest{
		Project:        projectID,
		Region:         region,
		ForwardingRule: ruleName,
	}

	op, err := client.Delete(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			log.Printf("Failed to find resource to delete: %v", err)
			return nil
		}

		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for forwarding rule deletion operation: %w", err)
	}

	log.Printf("Forwarding rule %s deleted successfully", ruleName)
	return nil
}
