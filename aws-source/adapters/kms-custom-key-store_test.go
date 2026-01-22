package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/overmindtech/cli/sdp-go"
)

func TestCustomKeyStoreOutputMapper(t *testing.T) {
	output := &kms.DescribeCustomKeyStoresOutput{
		CustomKeyStores: []types.CustomKeyStoresListEntry{
			{
				CustomKeyStoreId:       PtrString("custom-key-store-1"),
				CreationDate:           PtrTime(time.Now()),
				CloudHsmClusterId:      PtrString("cloud-hsm-cluster-1"),
				ConnectionState:        types.ConnectionStateTypeConnected,
				TrustAnchorCertificate: PtrString("-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwJ1z\n-----END CERTIFICATE-----"),
				CustomKeyStoreName:     PtrString("key-store-1"),
			},
		},
	}

	items, err := customKeyStoreOutputMapper(context.TODO(), nil, "foo", nil, output)
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

	item := items[0]

	tests := QueryTests{
		{
			ExpectedType:   "cloudhsmv2-cluster",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cloud-hsm-cluster-1",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewKMSCustomKeyStoreAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := kms.NewFromConfig(config)

	adapter := NewKMSCustomKeyStoreAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}

func TestHealthState(t *testing.T) {
	tests := []struct {
		name           string
		output         *kms.DescribeCustomKeyStoresOutput
		expectedHealth sdp.Health
		expectedError  error
	}{
		{
			name: "HealthyResourceReturnsHealthOK",
			output: &kms.DescribeCustomKeyStoresOutput{
				CustomKeyStores: []types.CustomKeyStoresListEntry{
					{
						CustomKeyStoreId: PtrString("custom-key-store-1"),
						ConnectionState:  types.ConnectionStateTypeConnected,
					},
				},
			},
			expectedHealth: sdp.Health_HEALTH_OK,
		},
		{
			name: "PendingResourceReturnsHealthPending",
			output: &kms.DescribeCustomKeyStoresOutput{
				CustomKeyStores: []types.CustomKeyStoresListEntry{
					{
						CustomKeyStoreId: PtrString("custom-key-store-1"),
						ConnectionState:  types.ConnectionStateTypeConnecting,
					},
				},
			},
			expectedHealth: sdp.Health_HEALTH_PENDING,
		},
		{
			name: "DisconnectedResourceReturnsHealthUnknown",
			output: &kms.DescribeCustomKeyStoresOutput{
				CustomKeyStores: []types.CustomKeyStoresListEntry{
					{
						CustomKeyStoreId: PtrString("custom-key-store-1"),
						ConnectionState:  types.ConnectionStateTypeDisconnected,
					},
				},
			},
			expectedHealth: sdp.Health_HEALTH_UNKNOWN,
		},
		{
			name: "FailedResourceReturnsHealthError",
			output: &kms.DescribeCustomKeyStoresOutput{
				CustomKeyStores: []types.CustomKeyStoresListEntry{
					{
						CustomKeyStoreId: PtrString("custom-key-store-1"),
						ConnectionState:  types.ConnectionStateTypeFailed,
					},
				},
			},
			expectedHealth: sdp.Health_HEALTH_ERROR,
		},
		{
			name: "UnknownConnectionStateReturnsError",
			output: &kms.DescribeCustomKeyStoresOutput{
				CustomKeyStores: []types.CustomKeyStoresListEntry{
					{
						CustomKeyStoreId: PtrString("custom-key-store-1"),
						ConnectionState:  "unknown-state",
					},
				},
			},
			expectedError: &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "unknown Connection State",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := customKeyStoreOutputMapper(context.TODO(), nil, "foo", nil, tt.output)
			if tt.expectedError != nil {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				if !errors.As(err, new(*sdp.QueryError)) {
					t.Errorf("expected %v, got %v", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != 1 {
					t.Fatalf("expected 1 item, got %v", len(items))
				}
				if items[0].GetHealth() != tt.expectedHealth {
					t.Errorf("expected health %v, got %v", tt.expectedHealth, items[0].GetHealth())
				}
			}
		})
	}
}
