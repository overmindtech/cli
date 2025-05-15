package example

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/example/shared"
)

func TestStaticData(t *testing.T) {
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Standard Wrapper", func(t *testing.T) {
		standardSearchableListable := NewStandardSearchableListable(nil, projectID, zone)

		adapter := sources.WrapperToAdapter(standardSearchableListable)

		if adapter.Type() != fmt.Sprintf("%s-%s-%s",
			shared.Source,
			shared.Compute,
			shared.Instance,
		) {
			t.Fatalf("Unexpected adapter type: %s", adapter.Type())
		}
		t.Logf("Adapter Type: type=%s", adapter.Type())

		if adapter.Name() != adapter.Type()+"-adapter" {
			t.Fatalf("Unexpected adapter name: %s", adapter.Name())
		}
		t.Logf("Adapter Name: name=%s", adapter.Name())

		if adapter.Scopes()[0] != fmt.Sprintf("%s.%s", projectID, zone) {
			t.Fatalf("Unexpected adapter scope: %s", adapter.Scopes()[0])
		}
		t.Logf("Adapter Scopes: scopes=%v", adapter.Scopes())

		metadata := adapter.Metadata()

		if metadata == nil {
			t.Fatalf("Adapter metadata is nil")
		}

		expectedDescriptiveName := fmt.Sprintf(
			"%s %s %s",
			strings.ToUpper(string(shared.Source)),
			cases.Title(language.English).String(string(shared.Compute)),
			cases.Title(language.English).String(string(shared.Instance)),
		)
		if metadata.GetDescriptiveName() != expectedDescriptiveName {
			t.Fatalf(
				"Unexpected adapter metadata descriptive name: %s, expected: %s",
				metadata.GetDescriptiveName(),
				expectedDescriptiveName,
			)
		}
		t.Logf("Metadata Descriptive Name: name=%s", metadata.GetDescriptiveName())

		if metadata.GetType() != adapter.Type() {
			t.Fatalf("Unexpected adapter metadata type: %s, expected: %s", metadata.GetType(), adapter.Type())
		}
		t.Logf("Metadata Type: type=%s", metadata.GetType())

		if metadata.GetCategory() != sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION {
			t.Fatalf("Unexpected adapter metadata category: %s", metadata.GetCategory())
		}
		t.Logf("Metadata Category: category=%s", metadata.GetCategory())

		tfMapping := metadata.GetTerraformMappings()[0]
		if tfMapping.GetTerraformMethod() != sdp.QueryMethod_GET {
			t.Fatalf("Expected TerraformMethod to be %s, but got: %s", sdp.QueryMethod_GET, tfMapping.GetTerraformMethod())
		}

		if tfMapping.GetTerraformQueryMap() != "example_resource.name" {
			t.Fatalf("Expected TerraformQueryMap to be 'example_resource.name', but got: %s", tfMapping.GetTerraformQueryMap())
		}
		t.Logf("Terraform QueryMap: mappings=%s", tfMapping.GetTerraformQueryMap())

		if !metadata.GetSupportedQueryMethods().GetGet() {
			t.Fatalf("Expected to support Get method")
		}

		expectedGetDescription := "Get GCP Compute Instance by \"GCP-compute-instance-id\""
		if metadata.GetSupportedQueryMethods().GetGetDescription() != expectedGetDescription {
			t.Fatalf("Expected GetDescription to be '%s', but got: %s", expectedGetDescription, metadata.GetSupportedQueryMethods().GetGetDescription())
		}
		t.Logf("Metadata GetDescription: description=%s", metadata.GetSupportedQueryMethods().GetGetDescription())

		if !metadata.GetSupportedQueryMethods().GetList() {
			t.Fatalf("Expected to support List method")
		}

		expectedListDescription := "List all GCP Compute Instance items"
		if metadata.GetSupportedQueryMethods().GetListDescription() != expectedListDescription {
			t.Fatalf("Expected ListDescription to be '%s', but got: %s", expectedListDescription, metadata.GetSupportedQueryMethods().GetListDescription())
		}
		t.Logf("Metadata ListDescription: description=%s", metadata.GetSupportedQueryMethods().GetListDescription())

		if !metadata.GetSupportedQueryMethods().GetSearch() {
			t.Fatalf("Expected to support Search method")
		}
		expectedSearchDescription := "Search for GCP Compute Instance by \"GCP-compute-status-id\" or \"GCP-compute-disk-name|GCP-compute-status-id\""
		if metadata.GetSupportedQueryMethods().GetSearchDescription() != expectedSearchDescription {
			t.Fatalf("Expected SearchDescription to be '%s', but got: %s", expectedSearchDescription, metadata.GetSupportedQueryMethods().GetSearchDescription())
		}
		t.Logf("Metadata SearchDescription: description=%s", metadata.GetSupportedQueryMethods().GetGetDescription())

		expectedPotentialLink := "GCP-compute-disk"
		potentialLink := metadata.GetPotentialLinks()[0]
		if potentialLink != expectedPotentialLink {
			t.Fatalf("Expected potential link to be %s, but got: %s", expectedPotentialLink, potentialLink)
		}
		t.Logf("Potential Links: links=%v", metadata.GetPotentialLinks())
	})

	t.Run("Custom Wrapper", func(t *testing.T) {
		customSearchableListable := NewCustomSearchableListable(nil, projectID, zone)

		adapter := sources.WrapperToAdapter(customSearchableListable)

		if adapter.Type() != fmt.Sprintf("%s-%s-%s",
			shared.Source,
			shared.Compute,
			shared.Instance,
		) {
			t.Fatalf("Unexpected adapter type: %s", adapter.Type())
		}
		t.Logf("Adapter Type: type=%s", adapter.Type())

		if adapter.Name() != adapter.Type()+"-adapter" {
			t.Fatalf("Unexpected adapter name: %s", adapter.Name())
		}
		t.Logf("Adapter Name: name=%s", adapter.Name())

		if adapter.Scopes()[0] != projectID {
			t.Fatalf("Unexpected adapter scope: %s", adapter.Scopes()[0])
		}

		if adapter.Scopes()[1] != fmt.Sprintf("%s.%s", projectID, zone) {
			t.Fatalf("Unexpected adapter scope: %s", adapter.Scopes()[0])
		}
		t.Logf("Adapter Scopes: scopes=%v", adapter.Scopes())

		metadata := adapter.Metadata()

		if metadata == nil {
			t.Fatalf("Adapter metadata is nil")
		}

		expectedDescriptiveName := "Custom descriptive name"
		if metadata.GetDescriptiveName() != expectedDescriptiveName {
			t.Fatalf(
				"Unexpected adapter metadata descriptive name: %s, expected: %s",
				metadata.GetDescriptiveName(),
				expectedDescriptiveName,
			)
		}
		t.Logf("Metadata Descriptive Name: name=%s", metadata.GetDescriptiveName())

		expectedGetDescription := "Get a compute instance by ID"
		if metadata.GetSupportedQueryMethods().GetGetDescription() != expectedGetDescription {
			t.Fatalf("Expected GetDescription to be '%s', but got: %s", expectedGetDescription, metadata.GetSupportedQueryMethods().GetGetDescription())
		}
		t.Logf("Metadata GetDescription: description=%s", metadata.GetSupportedQueryMethods().GetGetDescription())

		expectedListDescription := "List all compute instances"
		if metadata.GetSupportedQueryMethods().GetListDescription() != expectedListDescription {
			t.Fatalf("Expected ListDescription to be '%s', but got: %s", expectedListDescription, metadata.GetSupportedQueryMethods().GetListDescription())
		}
		t.Logf("Metadata ListDescription: description=%s", metadata.GetSupportedQueryMethods().GetListDescription())

		expectedSearchDescription := "Search for compute instances by {compute status id} or {compute disk name|compute status id}"
		if metadata.GetSupportedQueryMethods().GetSearchDescription() != expectedSearchDescription {
			t.Fatalf("Expected SearchDescription to be '%s', but got: %s", expectedSearchDescription, metadata.GetSupportedQueryMethods().GetSearchDescription())
		}
		t.Logf("Metadata SearchDescription: description=%s", metadata.GetSupportedQueryMethods().GetSearchDescription())

		if metadata.GetType() != adapter.Type() {
			t.Fatalf("Unexpected adapter metadata type: %s, expected: %s", metadata.GetType(), adapter.Type())
		}
		t.Logf("Metadata Type: type=%s", metadata.GetType())

		if metadata.GetCategory() != sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION {
			t.Fatalf("Unexpected adapter metadata category: %s", metadata.GetCategory())
		}
		t.Logf("Metadata Category: category=%s", metadata.GetCategory())

		tfMapping := metadata.GetTerraformMappings()[0]
		if tfMapping.GetTerraformMethod() != sdp.QueryMethod_GET {
			t.Fatalf("Expected TerraformMethod to be %s, but got: %s", sdp.QueryMethod_GET, tfMapping.GetTerraformMethod())
		}
	})
}
