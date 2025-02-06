package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestCustomerMetadataOutputMapper(t *testing.T) {
	output := &directconnect.DescribeCustomerMetadataOutput{
		Agreements: []types.CustomerAgreement{
			{
				AgreementName: adapterhelpers.PtrString("example-customer-agreement"),
				Status:        adapterhelpers.PtrString("signed"),
			},
		},
	}

	items, err := customerMetadataOutputMapper(context.Background(), nil, "foo", nil, output)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}
}

func TestNewDirectConnectCustomerMetadataAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectCustomerMetadataAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
