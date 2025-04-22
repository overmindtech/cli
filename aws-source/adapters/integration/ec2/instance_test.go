package ec2

import (
	"context"
	"fmt"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func searchSync(adapter discovery.StreamingAdapter, ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	stream := discovery.NewRecordingQueryResultStream()
	adapter.SearchStream(ctx, scope, query, ignoreCache, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to search: %v", errs)
	}

	return stream.GetItems(), nil
}

func listSync(adapter discovery.StreamingAdapter, ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(ctx, scope, ignoreCache, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to List: %v", errs)
	}

	return stream.GetItems(), nil
}

func EC2(t *testing.T) {
	ctx := context.Background()

	var err error
	testClient, err := ec2Client(ctx)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		t.Fatalf("Failed to get AWS settings: %v", err)
	}

	accountID := testAWSConfig.AccountID

	t.Log("Running EC2 integration test")

	instanceAdapter := adapters.NewEC2InstanceAdapter(testClient, accountID, testAWSConfig.Region)

	err = instanceAdapter.Validate()
	if err != nil {
		t.Fatalf("failed to validate EC2 instance adapter: %v", err)
	}

	scope := adapterhelpers.FormatScope(accountID, testAWSConfig.Region)

	// List instances
	sdpListInstances, err := listSync(instanceAdapter, context.Background(), scope, true)
	if err != nil {
		t.Fatalf("failed to list EC2 instances: %v", err)
	}

	if len(sdpListInstances) == 0 {
		t.Fatalf("no instances found")
	}

	uniqueAttribute := sdpListInstances[0].GetUniqueAttribute()

	instanceID, err := integration.GetUniqueAttributeValueByTags(
		uniqueAttribute,
		sdpListInstances,
		integration.ResourceTags(integration.EC2, instanceSrc),
		false,
	)
	if err != nil {
		t.Fatalf("failed to get instance ID: %v", err)
	}

	// Get instance
	sdpInstance, err := instanceAdapter.Get(context.Background(), scope, instanceID, true)
	if err != nil {
		t.Fatalf("failed to get EC2 instance: %v", err)
	}

	instanceIDFromGet, err := integration.GetUniqueAttributeValueByTags(uniqueAttribute, []*sdp.Item{sdpInstance}, integration.ResourceTags(integration.EC2, instanceSrc), false)
	if err != nil {
		t.Fatalf("failed to get instance ID from get: %v", err)
	}

	if instanceIDFromGet != instanceID {
		t.Fatalf("expected instance ID %v, got %v", instanceID, instanceIDFromGet)
	}

	// Search instances
	instanceARN := fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", testAWSConfig.Region, accountID, instanceID)
	sdpSearchInstances, err := searchSync(instanceAdapter, context.Background(), scope, instanceARN, true)
	if err != nil {
		t.Fatalf("failed to search EC2 instances: %v", err)
	}

	if len(sdpSearchInstances) == 0 {
		t.Fatalf("no instances found")
	}

	instanceIDFromSearch, err := integration.GetUniqueAttributeValueByTags(uniqueAttribute, sdpSearchInstances, integration.ResourceTags(integration.EC2, instanceSrc), false)
	if err != nil {
		t.Fatalf("failed to get instance ID from search: %v", err)
	}

	if instanceIDFromSearch != instanceID {
		t.Fatalf("expected instance ID %v, got %v", instanceID, instanceIDFromSearch)
	}
}
