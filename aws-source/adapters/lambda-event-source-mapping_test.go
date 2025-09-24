package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/overmindtech/cli/sdp-go"
)

type TestLambdaEventSourceMappingClient struct{}

func (t *TestLambdaEventSourceMappingClient) ListEventSourceMappings(ctx context.Context, params *lambda.ListEventSourceMappingsInput, optFns ...func(*lambda.Options)) (*lambda.ListEventSourceMappingsOutput, error) {
	allMappings := []types.EventSourceMappingConfiguration{
		{
			UUID:           stringPtr("test-uuid-1"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function"),
			EventSourceArn: stringPtr("arn:aws:sqs:us-east-1:123456789012:test-queue"),
			State:          stringPtr("Enabled"),
		},
		{
			UUID:           stringPtr("test-uuid-2"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-2"),
			EventSourceArn: stringPtr("arn:aws:dynamodb:us-east-1:123456789012:table/test-table"),
			State:          stringPtr("Creating"),
		},
		{
			UUID:           stringPtr("test-uuid-3"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-3"),
			EventSourceArn: stringPtr("arn:aws:rds:us-east-1:123456789012:cluster:test-docdb-cluster"),
			State:          stringPtr("Enabled"),
		},
	}

	// If EventSourceArn is specified, filter by it
	if params.EventSourceArn != nil {
		filtered := []types.EventSourceMappingConfiguration{}
		for _, mapping := range allMappings {
			if mapping.EventSourceArn != nil && *mapping.EventSourceArn == *params.EventSourceArn {
				filtered = append(filtered, mapping)
			}
		}
		return &lambda.ListEventSourceMappingsOutput{
			EventSourceMappings: filtered,
		}, nil
	}

	return &lambda.ListEventSourceMappingsOutput{
		EventSourceMappings: allMappings,
	}, nil
}

func (t *TestLambdaEventSourceMappingClient) GetEventSourceMapping(ctx context.Context, params *lambda.GetEventSourceMappingInput, optFns ...func(*lambda.Options)) (*lambda.GetEventSourceMappingOutput, error) {
	if params.UUID == nil {
		return nil, &types.ResourceNotFoundException{}
	}

	switch *params.UUID {
	case "test-uuid-1":
		return &lambda.GetEventSourceMappingOutput{
			UUID:           stringPtr("test-uuid-1"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function"),
			EventSourceArn: stringPtr("arn:aws:sqs:us-east-1:123456789012:test-queue"),
			State:          stringPtr("Enabled"),
		}, nil
	case "test-uuid-2":
		return &lambda.GetEventSourceMappingOutput{
			UUID:           stringPtr("test-uuid-2"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-2"),
			EventSourceArn: stringPtr("arn:aws:dynamodb:us-east-1:123456789012:table/test-table"),
			State:          stringPtr("Creating"),
		}, nil
	case "test-uuid-3":
		return &lambda.GetEventSourceMappingOutput{
			UUID:           stringPtr("test-uuid-3"),
			FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-3"),
			EventSourceArn: stringPtr("arn:aws:rds:us-east-1:123456789012:cluster:test-docdb-cluster"),
			State:          stringPtr("Enabled"),
		}, nil
	default:
		return nil, &types.ResourceNotFoundException{}
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestLambdaEventSourceMappingAdapter(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test adapter metadata
	if adapter.Type() != "lambda-event-source-mapping" {
		t.Errorf("Expected adapter type to be 'lambda-event-source-mapping', got %s", adapter.Type())
	}

	if adapter.Name() != "lambda-event-source-mapping-adapter" {
		t.Errorf("Expected adapter name to be 'lambda-event-source-mapping-adapter', got %s", adapter.Name())
	}

	// Test scopes
	scopes := adapter.Scopes()
	if len(scopes) != 1 {
		t.Errorf("Expected 1 scope, got %d", len(scopes))
	}
	if scopes[0] != "123456789012.us-east-1" {
		t.Errorf("Expected scope to be '123456789012.us-east-1', got %s", scopes[0])
	}
}

func TestLambdaEventSourceMappingGetFunc(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test getting existing event source mapping
	item, err := adapter.Get(context.Background(), "123456789012.us-east-1", "test-uuid-1", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if item == nil {
		t.Error("Expected item, got nil")
		return
	}

	if item.GetType() != "lambda-event-source-mapping" {
		t.Errorf("Expected item type to be 'lambda-event-source-mapping', got %s", item.GetType())
	}

	if uuid, _ := item.GetAttributes().Get("UUID"); uuid != "test-uuid-1" {
		t.Errorf("Expected UUID to be 'test-uuid-1', got %s", uuid)
	}

	// Test getting non-existent event source mapping
	_, err = adapter.Get(context.Background(), "123456789012.us-east-1", "non-existent-uuid", false)
	if err == nil {
		t.Error("Expected error for non-existent UUID, got nil")
	}

	// Test wrong scope
	_, err = adapter.Get(context.Background(), "wrong-scope", "test-uuid-1", false)
	if err == nil {
		t.Error("Expected error for wrong scope, got nil")
	}
}

func TestLambdaEventSourceMappingItemMapper(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test mapping with SQS event source
	awsItem := &types.EventSourceMappingConfiguration{
		UUID:           stringPtr("test-uuid-1"),
		FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function"),
		EventSourceArn: stringPtr("arn:aws:sqs:us-east-1:123456789012:test-queue"),
		State:          stringPtr("Enabled"),
	}

	item, err := adapter.ItemMapper("test-uuid-1", "123456789012.us-east-1", awsItem)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if item.GetType() != "lambda-event-source-mapping" {
		t.Errorf("Expected item type to be 'lambda-event-source-mapping', got %s", item.GetType())
	}

	if uuid, _ := item.GetAttributes().Get("UUID"); uuid != "test-uuid-1" {
		t.Errorf("Expected UUID to be 'test-uuid-1', got %s", uuid)
	}

	if functionArn, _ := item.GetAttributes().Get("FunctionArn"); functionArn != "arn:aws:lambda:us-east-1:123456789012:function:test-function" {
		t.Errorf("Expected FunctionArn to match, got %s", functionArn)
	}

	if eventSourceArn, _ := item.GetAttributes().Get("EventSourceArn"); eventSourceArn != "arn:aws:sqs:us-east-1:123456789012:test-queue" {
		t.Errorf("Expected EventSourceArn to match, got %s", eventSourceArn)
	}

	// Check health status
	if item.Health == nil {
		t.Error("Expected health to be set")
	} else if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Errorf("Expected health to be HEALTH_OK, got %v", item.GetHealth())
	}

	// Check linked items
	if len(item.GetLinkedItemQueries()) != 2 {
		t.Errorf("Expected 2 linked items, got %d", len(item.GetLinkedItemQueries()))
	}

	// Check Lambda function link
	lambdaLink := item.GetLinkedItemQueries()[0]
	if lambdaLink.GetQuery().GetType() != "lambda-function" {
		t.Errorf("Expected Lambda function link type to be 'lambda-function', got %s", lambdaLink.GetQuery().GetType())
	}
	if lambdaLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected Lambda function link method to be SEARCH, got %v", lambdaLink.GetQuery().GetMethod())
	}

	// Check SQS queue link
	sqsLink := item.GetLinkedItemQueries()[1]
	if sqsLink.GetQuery().GetType() != "sqs-queue" {
		t.Errorf("Expected SQS queue link type to be 'sqs-queue', got %s", sqsLink.GetQuery().GetType())
	}
	if sqsLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected SQS queue link method to be SEARCH, got %v", sqsLink.GetQuery().GetMethod())
	}
}

func TestLambdaEventSourceMappingItemMapperWithDynamoDB(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test mapping with DynamoDB event source
	awsItem := &types.EventSourceMappingConfiguration{
		UUID:           stringPtr("test-uuid-2"),
		FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-2"),
		EventSourceArn: stringPtr("arn:aws:dynamodb:us-east-1:123456789012:table/test-table"),
		State:          stringPtr("Creating"),
	}

	item, err := adapter.ItemMapper("test-uuid-2", "123456789012.us-east-1", awsItem)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check DynamoDB table link
	dynamoLink := item.GetLinkedItemQueries()[1]
	if dynamoLink.GetQuery().GetType() != "dynamodb-table" {
		t.Errorf("Expected DynamoDB table link type to be 'dynamodb-table', got %s", dynamoLink.GetQuery().GetType())
	}
	if dynamoLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected DynamoDB table link method to be SEARCH, got %v", dynamoLink.GetQuery().GetMethod())
	}

	// Check health status for Creating state
	if item.Health == nil {
		t.Error("Expected health to be set")
	} else if item.GetHealth() != sdp.Health_HEALTH_PENDING {
		t.Errorf("Expected health to be HEALTH_PENDING, got %v", item.GetHealth())
	}
}

func TestLambdaEventSourceMappingItemMapperWithRDS(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test mapping with RDS/DocumentDB event source
	awsItem := &types.EventSourceMappingConfiguration{
		UUID:           stringPtr("test-uuid-3"),
		FunctionArn:    stringPtr("arn:aws:lambda:us-east-1:123456789012:function:test-function-3"),
		EventSourceArn: stringPtr("arn:aws:rds:us-east-1:123456789012:cluster:test-docdb-cluster"),
		State:          stringPtr("Enabled"),
	}

	item, err := adapter.ItemMapper("test-uuid-3", "123456789012.us-east-1", awsItem)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check RDS cluster link
	rdsLink := item.GetLinkedItemQueries()[1]
	if rdsLink.GetQuery().GetType() != "rds-db-cluster" {
		t.Errorf("Expected RDS cluster link type to be 'rds-db-cluster', got %s", rdsLink.GetQuery().GetType())
	}
	if rdsLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected RDS cluster link method to be SEARCH, got %v", rdsLink.GetQuery().GetMethod())
	}

	// Check health status
	if item.Health == nil {
		t.Error("Expected health to be set")
	} else if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Errorf("Expected health to be HEALTH_OK, got %v", item.GetHealth())
	}
}

func TestLambdaEventSourceMappingSearchByEventSourceARN(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test search by SQS queue ARN
	sqsQueueARN := "arn:aws:sqs:us-east-1:123456789012:test-queue"
	items, err := adapter.Search(context.Background(), "123456789012.us-east-1", sqsQueueARN, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	// The item should have the correct event source ARN
	if eventSourceArn, _ := items[0].GetAttributes().Get("EventSourceArn"); eventSourceArn != sqsQueueARN {
		t.Errorf("Expected EventSourceArn '%s', got '%s'", sqsQueueARN, eventSourceArn)
	}
}

func TestLambdaEventSourceMappingSearchWrongScope(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test search with wrong scope
	_, err := adapter.Search(context.Background(), "wrong-scope", "arn:aws:sqs:us-east-1:123456789012:test-queue", false)
	if err == nil {
		t.Error("Expected error for wrong scope, got nil")
	}
}

func TestLambdaEventSourceMappingAdapterList(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test List
	items, err := adapter.List(context.Background(), "123456789012.us-east-1", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	// Verify we get all the expected items
	expectedUUIDs := []string{"test-uuid-1", "test-uuid-2", "test-uuid-3"}
	foundUUIDs := make(map[string]bool)

	for _, item := range items {
		if uuid, _ := item.GetAttributes().Get("UUID"); uuid != nil {
			foundUUIDs[uuid.(string)] = true
		}

		if item.GetType() != "lambda-event-source-mapping" {
			t.Errorf("Expected item type to be 'lambda-event-source-mapping', got %s", item.GetType())
		}
	}

	for _, expectedUUID := range expectedUUIDs {
		if !foundUUIDs[expectedUUID] {
			t.Errorf("Expected to find UUID %s in list results", expectedUUID)
		}
	}
}

func TestLambdaEventSourceMappingAdapterListWrongScope(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test List with wrong scope
	_, err := adapter.List(context.Background(), "wrong-scope", false)
	if err == nil {
		t.Error("Expected error for wrong scope, got nil")
	}
}

func TestLambdaEventSourceMappingAdapterIntegration(t *testing.T) {
	adapter := NewLambdaEventSourceMappingAdapter(&TestLambdaEventSourceMappingClient{}, "123456789012", "us-east-1")

	// Test Get
	item, err := adapter.Get(context.Background(), "123456789012.us-east-1", "test-uuid-1", false)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if item == nil {
		t.Error("Get returned nil item")
	}

	// Test List
	items, err := adapter.List(context.Background(), "123456789012.us-east-1", false)
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("Expected 3 items from list, got %d", len(items))
	}

	// Test Search by event source ARN
	sqsQueueARN := "arn:aws:sqs:us-east-1:123456789012:test-queue"
	searchItems, err := adapter.Search(context.Background(), "123456789012.us-east-1", sqsQueueARN, false)
	if err != nil {
		t.Errorf("Search by event source ARN failed: %v", err)
	}
	if len(searchItems) != 1 {
		t.Errorf("Expected 1 item from search, got %d", len(searchItems))
	}
}
