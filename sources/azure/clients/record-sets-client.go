package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

//go:generate mockgen -destination=../shared/mocks/mock_record_sets_client.go -package=mocks -source=record-sets-client.go

// RecordSetsPager is a type alias for the generic Pager interface with record sets list response type.
type RecordSetsPager = Pager[armdns.RecordSetsClientListAllByDNSZoneResponse]

// RecordSetsClient is an interface for interacting with Azure DNS record sets
type RecordSetsClient interface {
	Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType armdns.RecordType, options *armdns.RecordSetsClientGetOptions) (armdns.RecordSetsClientGetResponse, error)
	NewListAllByDNSZonePager(resourceGroupName string, zoneName string, options *armdns.RecordSetsClientListAllByDNSZoneOptions) RecordSetsPager
}

type recordSetsClient struct {
	client *armdns.RecordSetsClient
}

func (c *recordSetsClient) Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType armdns.RecordType, options *armdns.RecordSetsClientGetOptions) (armdns.RecordSetsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, zoneName, relativeRecordSetName, recordType, options)
}

func (c *recordSetsClient) NewListAllByDNSZonePager(resourceGroupName string, zoneName string, options *armdns.RecordSetsClientListAllByDNSZoneOptions) RecordSetsPager {
	return c.client.NewListAllByDNSZonePager(resourceGroupName, zoneName, options)
}

// NewRecordSetsClient creates a new RecordSetsClient from the Azure SDK client
func NewRecordSetsClient(client *armdns.RecordSetsClient) RecordSetsClient {
	return &recordSetsClient{client: client}
}
