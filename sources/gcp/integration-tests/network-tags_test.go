// Run commands (assumes RUN_GCP_INTEGRATION_TESTS, GCP_PROJECT_ID, GCP_ZONE are exported):
//
//   All:      go test ./sources/gcp/integration-tests/ -run "TestNetworkTagRelationships" -count 1 -v
//   Setup:    go test ./sources/gcp/integration-tests/ -run "TestNetworkTagRelationships/Setup" -count 1 -v
//   Run:      go test ./sources/gcp/integration-tests/ -run "TestNetworkTagRelationships/(Instance|Firewall|Route)" -count 1 -v
//   Teardown: go test ./sources/gcp/integration-tests/ -run "TestNetworkTagRelationships/Teardown" -count 1 -v
//
// Verify created resources with gcloud:
//
//   gcloud compute instances describe integration-test-nettag-instance --zone=$GCP_ZONE --project=$GCP_PROJECT_ID --format="value(tags.items)"
//   gcloud compute firewall-rules describe integration-test-nettag-fw --project=$GCP_PROJECT_ID --format="value(targetTags)"
//   gcloud compute routes describe integration-test-nettag-route --project=$GCP_PROJECT_ID --format="value(tags)"
//

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

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

const (
	networkTagTestInstance         = "integration-test-nettag-instance"
	networkTagTestFirewall         = "integration-test-nettag-fw"
	networkTagTestRoute            = "integration-test-nettag-route"
	networkTagTestInstanceTemplate = "integration-test-nettag-template"
	networkTag                     = "nettag-test"
)

func TestNetworkTagRelationships(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	t.Parallel()

	ctx := context.Background()

	instanceClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstancesRESTClient: %v", err)
	}
	defer instanceClient.Close()

	firewallClient, err := compute.NewFirewallsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewFirewallsRESTClient: %v", err)
	}
	defer firewallClient.Close()

	routeClient, err := compute.NewRoutesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewRoutesRESTClient: %v", err)
	}
	defer routeClient.Close()

	instanceTemplateClient, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstanceTemplatesRESTClient: %v", err)
	}
	defer instanceTemplateClient.Close()

	// --- Setup ---
	t.Run("Setup", func(t *testing.T) {
		if err := createInstanceWithTags(ctx, instanceClient, projectID, zone); err != nil {
			t.Fatalf("Failed to create tagged instance: %v", err)
		}
		if err := createFirewallWithTags(ctx, firewallClient, projectID); err != nil {
			t.Fatalf("Failed to create tagged firewall: %v", err)
		}
		if err := createRouteWithTags(ctx, routeClient, projectID); err != nil {
			t.Fatalf("Failed to create tagged route: %v", err)
		}
		if err := createInstanceTemplateWithTags(ctx, instanceTemplateClient, projectID); err != nil {
			t.Fatalf("Failed to create tagged instance template: %v", err)
		}
	})

	// --- Run ---
	t.Run("InstanceEmitsSearchLinksToFirewallAndRoute", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(
			gcpshared.NewComputeInstanceClient(instanceClient),
			[]gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)},
		)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], networkTagTestInstance, true)
		if qErr != nil {
			t.Fatalf("Get instance: %v", qErr)
		}

		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeFirewall.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeRoute.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
	})

	t.Run("FirewallSearchByTagReturnsFirewall", func(t *testing.T) {
		gcpHTTPCli, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
		if err != nil {
			t.Fatalf("GCPHTTPClientWithOtel: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeFirewall, gcpshared.NewLinker(), gcpHTTPCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("MakeAdapter: %v", err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Firewall adapter does not implement SearchableAdapter")
		}

		items, qErr := searchable.Search(ctx, projectID, networkTag, true)
		if qErr != nil {
			t.Fatalf("Search: %v", qErr)
		}

		found := false
		for _, item := range items {
			if v, err := item.GetAttributes().Get("name"); err == nil && v == networkTagTestFirewall {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find firewall %s in search results for tag %q, got %d items", networkTagTestFirewall, networkTag, len(items))
		}
	})

	t.Run("RouteSearchByTagReturnsRoute", func(t *testing.T) {
		gcpHTTPCli, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
		if err != nil {
			t.Fatalf("GCPHTTPClientWithOtel: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeRoute, gcpshared.NewLinker(), gcpHTTPCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("MakeAdapter: %v", err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Route adapter does not implement SearchableAdapter")
		}

		items, qErr := searchable.Search(ctx, projectID, networkTag, true)
		if qErr != nil {
			t.Fatalf("Search: %v", qErr)
		}

		found := false
		for _, item := range items {
			if v, err := item.GetAttributes().Get("name"); err == nil && v == networkTagTestRoute {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find route %s in search results for tag %q, got %d items", networkTagTestRoute, networkTag, len(items))
		}
	})

	t.Run("InstanceSearchByTagReturnsInstance", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(
			gcpshared.NewComputeInstanceClient(instanceClient),
			[]gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)},
		)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Instance adapter does not implement SearchableAdapter")
		}

		scopeWithZone := fmt.Sprintf("%s.%s", projectID, zone)
		items, qErr := searchable.Search(ctx, scopeWithZone, networkTag, true)
		if qErr != nil {
			t.Fatalf("Search: %v", qErr)
		}

		found := false
		for _, item := range items {
			if v, err := item.GetAttributes().Get("name"); err == nil && v == networkTagTestInstance {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find instance %s in search results for tag %q, got %d items", networkTagTestInstance, networkTag, len(items))
		}
	})

	t.Run("FirewallEmitsSearchLinksToInstance", func(t *testing.T) {
		gcpHTTPCli, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
		if err != nil {
			t.Fatalf("GCPHTTPClientWithOtel: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeFirewall, gcpshared.NewLinker(), gcpHTTPCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("MakeAdapter: %v", err)
		}

		sdpItem, qErr := adapter.Get(ctx, projectID, networkTagTestFirewall, true)
		if qErr != nil {
			t.Fatalf("Get firewall: %v", qErr)
		}

		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeInstance.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
	})

	t.Run("RouteEmitsSearchLinksToInstance", func(t *testing.T) {
		gcpHTTPCli, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
		if err != nil {
			t.Fatalf("GCPHTTPClientWithOtel: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeRoute, gcpshared.NewLinker(), gcpHTTPCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("MakeAdapter: %v", err)
		}

		sdpItem, qErr := adapter.Get(ctx, projectID, networkTagTestRoute, true)
		if qErr != nil {
			t.Fatalf("Get route: %v", qErr)
		}

		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeInstance.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
	})

	t.Run("InstanceTemplateEmitsSearchLinksToFirewallAndRoute", func(t *testing.T) {
		gcpHTTPCli, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
		if err != nil {
			t.Fatalf("GCPHTTPClientWithOtel: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeInstanceTemplate, gcpshared.NewLinker(), gcpHTTPCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("MakeAdapter: %v", err)
		}

		sdpItem, qErr := adapter.Get(ctx, projectID, networkTagTestInstanceTemplate, true)
		if qErr != nil {
			t.Fatalf("Get instance template: %v", qErr)
		}

		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeFirewall.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
		assertHasLinkedItemQuery(t, sdpItem, gcpshared.ComputeRoute.String(), sdp.QueryMethod_SEARCH, networkTag, projectID)
	})

	// --- Teardown ---
	t.Run("Teardown", func(t *testing.T) {
		if err := deleteComputeInstance(ctx, instanceClient, projectID, zone, networkTagTestInstance); err != nil {
			t.Errorf("Failed to delete instance: %v", err)
		}
		if err := deleteFirewall(ctx, firewallClient, projectID, networkTagTestFirewall); err != nil {
			t.Errorf("Failed to delete firewall: %v", err)
		}
		if err := deleteRoute(ctx, routeClient, projectID, networkTagTestRoute); err != nil {
			t.Errorf("Failed to delete route: %v", err)
		}
		if err := deleteInstanceTemplate(ctx, instanceTemplateClient, projectID, networkTagTestInstanceTemplate); err != nil {
			t.Errorf("Failed to delete instance template: %v", err)
		}
	})
}

func assertHasLinkedItemQuery(t *testing.T, item *sdp.Item, expectedType string, expectedMethod sdp.QueryMethod, expectedQuery, expectedScope string) {
	t.Helper()
	for _, liq := range item.GetLinkedItemQueries() {
		q := liq.GetQuery()
		if q.GetType() == expectedType && q.GetMethod() == expectedMethod && q.GetQuery() == expectedQuery && q.GetScope() == expectedScope {
			return
		}
	}
	t.Errorf("Missing LinkedItemQuery{type=%s, method=%s, query=%s, scope=%s} on item %s",
		expectedType, expectedMethod, expectedQuery, expectedScope, item.UniqueAttributeValue())
}

// --- Resource creation/deletion helpers ---

func createInstanceWithTags(ctx context.Context, client *compute.InstancesClient, projectID, zone string) error {
	instance := &computepb.Instance{
		Name:        new(networkTagTestInstance),
		MachineType: new(fmt.Sprintf("zones/%s/machineTypes/e2-micro", zone)),
		Tags: &computepb.Tags{
			Items: []string{networkTag},
		},
		Disks: []*computepb.AttachedDisk{
			{
				Boot:       new(true),
				AutoDelete: new(true),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					SourceImage: new("projects/debian-cloud/global/images/debian-12-bookworm-v20250415"),
					DiskSizeGb:  new(int64(10)),
				},
			},
		},
		NetworkInterfaces: []*computepb.NetworkInterface{
			{StackType: new("IPV4_ONLY")},
		},
	}

	op, err := client.Insert(ctx, &computepb.InsertInstanceRequest{
		Project:          projectID,
		Zone:             zone,
		InstanceResource: instance,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Instance %s already exists, skipping", networkTagTestInstance)
			return nil
		}
		return fmt.Errorf("insert instance: %w", err)
	}
	return op.Wait(ctx)
}

func createFirewallWithTags(ctx context.Context, client *compute.FirewallsClient, projectID string) error {
	fw := &computepb.Firewall{
		Name:       new(networkTagTestFirewall),
		Network:    new(fmt.Sprintf("projects/%s/global/networks/default", projectID)),
		TargetTags: []string{networkTag},
		Allowed: []*computepb.Allowed{
			{
				IPProtocol: new("tcp"),
				Ports:      []string{"8080"},
			},
		},
		SourceRanges: []string{"0.0.0.0/0"},
	}

	op, err := client.Insert(ctx, &computepb.InsertFirewallRequest{
		Project:          projectID,
		FirewallResource: fw,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Firewall %s already exists, skipping", networkTagTestFirewall)
			return nil
		}
		return fmt.Errorf("insert firewall: %w", err)
	}
	return op.Wait(ctx)
}

func createRouteWithTags(ctx context.Context, client *compute.RoutesClient, projectID string) error {
	route := &computepb.Route{
		Name:           new(networkTagTestRoute),
		Network:        new(fmt.Sprintf("projects/%s/global/networks/default", projectID)),
		DestRange:      new("10.99.0.0/24"),
		NextHopGateway: new(fmt.Sprintf("projects/%s/global/gateways/default-internet-gateway", projectID)),
		Tags:           []string{networkTag},
		Priority:       new(uint32(900)),
	}

	op, err := client.Insert(ctx, &computepb.InsertRouteRequest{
		Project:       projectID,
		RouteResource: route,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Route %s already exists, skipping", networkTagTestRoute)
			return nil
		}
		return fmt.Errorf("insert route: %w", err)
	}
	return op.Wait(ctx)
}

func deleteFirewall(ctx context.Context, client *compute.FirewallsClient, projectID, name string) error {
	op, err := client.Delete(ctx, &computepb.DeleteFirewallRequest{
		Project:  projectID,
		Firewall: name,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("delete firewall: %w", err)
	}
	return op.Wait(ctx)
}

func deleteRoute(ctx context.Context, client *compute.RoutesClient, projectID, name string) error {
	op, err := client.Delete(ctx, &computepb.DeleteRouteRequest{
		Project: projectID,
		Route:   name,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("delete route: %w", err)
	}
	return op.Wait(ctx)
}

func createInstanceTemplateWithTags(ctx context.Context, client *compute.InstanceTemplatesClient, projectID string) error {
	template := &computepb.InstanceTemplate{
		Name: new(networkTagTestInstanceTemplate),
		Properties: &computepb.InstanceProperties{
			MachineType: new("e2-micro"),
			Tags: &computepb.Tags{
				Items: []string{networkTag},
			},
			Disks: []*computepb.AttachedDisk{
				{
					Boot:       new(true),
					AutoDelete: new(true),
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						SourceImage: new("projects/debian-cloud/global/images/debian-12-bookworm-v20250415"),
						DiskSizeGb:  new(int64(10)),
					},
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Network:   new("global/networks/default"),
					StackType: new("IPV4_ONLY"),
				},
			},
		},
	}

	op, err := client.Insert(ctx, &computepb.InsertInstanceTemplateRequest{
		Project:                  projectID,
		InstanceTemplateResource: template,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Instance template %s already exists, skipping", networkTagTestInstanceTemplate)
			return nil
		}
		return fmt.Errorf("insert instance template: %w", err)
	}
	return op.Wait(ctx)
}
