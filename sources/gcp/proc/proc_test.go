package proc

import (
	"context"
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func Test_adapters(t *testing.T) {
	ctx := context.Background()
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("error creating adapters: %v", err)
	}

	numberOfAdapters := len(discoveryAdapters)

	if numberOfAdapters == 0 {
		t.Fatal("Expected at least one adapter, got none")
	}

	if len(Metadata.AllAdapterMetadata()) != numberOfAdapters {
		t.Fatalf("Expected %d adapters in metadata, got %d", numberOfAdapters, len(Metadata.AllAdapterMetadata()))
	}

	// Check if the Spanner adapter is present
	// Because it is created externally and it needs to be registered during the initialization of the source
	// we need to ensure that it is included in the discoveryAdapters list.
	spannerAdapterFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.SpannerDatabase.String() {
			spannerAdapterFound = true
			break
		}
	}

	if !spannerAdapterFound {
		t.Fatal("Expected to find Spanner adapter in the list of adapters")
	}

	aiPlatformCustomJobFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.AIPlatformCustomJob.String() {
			aiPlatformCustomJobFound = true
			break
		}
	}

	if !aiPlatformCustomJobFound {
		t.Fatal("Expected to find AIPlatform Custom Job adapter in the list of adapters")
	}

	t.Logf("GCP Adapters found: %v", len(discoveryAdapters))
}
