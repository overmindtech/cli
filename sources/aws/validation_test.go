package aws

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
)

type Validate interface {
	Validate() error
}

func TestAdaptersValidation(t *testing.T) {
	accountID := "123456789012"
	region := "us-east-1"

	var adapters []discovery.Adapter
	adapters = append(adapters,
		sources.WrapperToAdapter(NewAPIGatewayStage(nil, accountID, region)),
		sources.WrapperToAdapter(NewApiGatewayAPIKey(nil, accountID, region)),
	)

	for _, adapter := range adapters {
		t.Run(adapter.Name(), func(t *testing.T) {
			// Test the adapter
			a, ok := adapter.(Validate)
			if !ok {
				t.Fatalf("Adapter %s does not implement Validate", adapter.Name())
			}

			if err := a.Validate(); err != nil {
				t.Fatalf("Adapter %s failed validation: %v", adapter.Name(), err)
			}

			if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
				// Pretty print the adapter metadata via json
				jsonData, err := json.MarshalIndent(adapter.Metadata(), "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal adapter metadata: %v", err)
				}
				t.Logf("Adapter %s metadata: %s", adapter.Name(), string(jsonData))
			}
		})
	}
}
