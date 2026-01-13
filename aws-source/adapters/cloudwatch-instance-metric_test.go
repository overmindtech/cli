package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdpcache"
)

// testCloudwatchMetricClient is a mock client for testing GetMetricData
type testCloudwatchMetricClient struct{}

func (c testCloudwatchMetricClient) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	now := time.Now()
	// Return data for all metrics
	results := make([]types.MetricDataResult, 0, len(ec2InstanceMetrics))
	for i, metricName := range ec2InstanceMetrics {
		// Each metric gets a single value (15-minute average)
		value := 50.0 + float64(i)*5.0 // Different values for each metric
		var id string
		if params.MetricDataQueries[i].Id != nil {
			id = *params.MetricDataQueries[i].Id
		} else {
			id = fmt.Sprintf("m%d", i)
		}
		results = append(results, types.MetricDataResult{
			Id:         aws.String(id),
			Label:      aws.String(metricName),
			Timestamps: []time.Time{now},
			Values:     []float64{value},
			StatusCode: types.StatusCodeComplete,
		})
	}
	return &cloudwatch.GetMetricDataOutput{
		MetricDataResults: results,
		Messages:          []types.MessageData{},
	}, nil
}

// testCloudwatchMetricClientEmpty returns no data
type testCloudwatchMetricClientEmpty struct{}

func (c testCloudwatchMetricClientEmpty) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{},
		Messages:          []types.MessageData{},
	}, nil
}

// testCloudwatchMetricClientWithCallCount tracks how many times GetMetricData is called
type testCloudwatchMetricClientWithCallCount struct {
	callCount int
}

func (c *testCloudwatchMetricClientWithCallCount) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	c.callCount++
	now := time.Now()
	// Return data for all metrics
	results := make([]types.MetricDataResult, 0, len(ec2InstanceMetrics))
	for i, metricName := range ec2InstanceMetrics {
		// Each metric gets a single value (15-minute average)
		value := 50.0 + float64(i)*5.0 // Different values for each metric
		var id string
		if params.MetricDataQueries[i].Id != nil {
			id = *params.MetricDataQueries[i].Id
		} else {
			id = fmt.Sprintf("m%d", i)
		}
		results = append(results, types.MetricDataResult{
			Id:         aws.String(id),
			Label:      aws.String(metricName),
			Timestamps: []time.Time{now},
			Values:     []float64{value},
			StatusCode: types.StatusCodeComplete,
		})
	}
	return &cloudwatch.GetMetricDataOutput{
		MetricDataResults: results,
		Messages:          []types.MessageData{},
	}, nil
}

func TestValidateInstanceID(t *testing.T) {
	tests := []struct {
		name        string
		instanceID  string
		expectError bool
	}{
		{
			name:        "valid instance ID - 17 characters (newer format)",
			instanceID:  "i-1234567890abcdef0",
			expectError: false,
		},
		{
			name:        "valid instance ID - 8 characters (older format)",
			instanceID:  "i-12345678",
			expectError: false,
		},
		{
			name:        "invalid format - missing i-",
			instanceID:  "1234567890abcdef0",
			expectError: true,
		},
		{
			name:        "invalid format - too short",
			instanceID:  "i-1234567",
			expectError: true,
		},
		{
			name:        "invalid format - wrong length (9 characters)",
			instanceID:  "i-123456789",
			expectError: true,
		},
		{
			name:        "invalid format - too long",
			instanceID:  "i-1234567890abcdef01",
			expectError: true,
		},
		{
			name:        "invalid format - invalid characters",
			instanceID:  "i-1234567890abcdefg",
			expectError: true,
		},
		{
			name:        "empty string",
			instanceID:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInstanceID(tt.instanceID)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMetricOutputMapper(t *testing.T) {
	ctx := context.Background()
	client := testCloudwatchMetricClient{}
	scope := "123456789012.eu-west-2"
	instanceID := "i-1234567890abcdef0"

	now := time.Now()
	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			{
				Id:         aws.String("m0"),
				Label:      aws.String("CPUUtilization"),
				Timestamps: []time.Time{now},
				Values:     []float64{45.5},
				StatusCode: types.StatusCodeComplete,
			},
			{
				Id:         aws.String("m1"),
				Label:      aws.String("NetworkIn"),
				Timestamps: []time.Time{now},
				Values:     []float64{1024.0},
				StatusCode: types.StatusCodeComplete,
			},
		},
		Messages: []types.MessageData{},
	}

	item, err := metricOutputMapper(ctx, client, scope, instanceID, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = item.Validate(); err != nil {
		t.Errorf("item validation failed: %v", err)
	}

	// Check type and unique attribute
	if item.GetType() != "cloudwatch-instance-metric" {
		t.Errorf("expected type cloudwatch-instance-metric, got %s", item.GetType())
	}
	if item.GetUniqueAttribute() != "InstanceId" {
		t.Errorf("expected unique attribute InstanceId, got %s", item.GetUniqueAttribute())
	}
	if item.GetScope() != scope {
		t.Errorf("expected scope %s, got %s", scope, item.GetScope())
	}

	// Check attributes
	attrs := item.GetAttributes()
	if attrs == nil {
		t.Fatal("attributes are nil")
	}

	// Verify key attributes exist
	attrMap := attrs.GetAttrStruct().AsMap()

	if attrMap["InstanceId"] != instanceID {
		t.Errorf("expected InstanceId %s, got %v", instanceID, attrMap["InstanceId"])
	}
	if attrMap["DataAvailable"] != true {
		t.Errorf("expected DataAvailable true, got %v", attrMap["DataAvailable"])
	}
	if attrMap["CPUUtilization"].(float64) != 45.5 {
		t.Errorf("expected CPUUtilization 45.5, got %v", attrMap["CPUUtilization"])
	}
	if attrMap["CPUUtilization_Formatted"] != "45.50%" {
		t.Errorf("expected CPUUtilization_Formatted '45.50%%', got %v", attrMap["CPUUtilization_Formatted"])
	}
	if attrMap["NetworkIn"].(float64) != 1024.0 {
		t.Errorf("expected NetworkIn 1024.0, got %v", attrMap["NetworkIn"])
	}
	if attrMap["NetworkIn_Formatted"] != "1.00 KB/s" {
		t.Errorf("expected NetworkIn_Formatted '1.00 KB/s', got %v", attrMap["NetworkIn_Formatted"])
	}

	// Verify metadata about the averaging period
	if attrMap["Statistic"] != "Average" {
		t.Errorf("expected Statistic 'Average', got %v", attrMap["Statistic"])
	}
	if attrMap["PeriodMinutes"].(float64) != 15 {
		t.Errorf("expected PeriodMinutes 15, got %v", attrMap["PeriodMinutes"])
	}
}

func TestMetricOutputMapperNoData(t *testing.T) {
	ctx := context.Background()
	client := testCloudwatchMetricClientEmpty{}
	scope := "123456789012.eu-west-2"
	instanceID := "i-1234567890abcdef0"

	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{},
		Messages:          []types.MessageData{},
	}

	item, err := metricOutputMapper(ctx, client, scope, instanceID, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = item.Validate(); err != nil {
		t.Errorf("item validation failed: %v", err)
	}

	attrMap := item.GetAttributes().GetAttrStruct().AsMap()

	// Should indicate no data available
	if attrMap["DataAvailable"] != false {
		t.Errorf("expected DataAvailable false, got %v", attrMap["DataAvailable"])
	}
}

func TestCloudwatchInstanceMetricAdapterGet(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scope := "123456789012.eu-west-2"
	query := "i-1234567890abcdef0"

	item, err := adapter.Get(context.Background(), scope, query, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item == nil {
		t.Fatal("expected item, got nil")
	}

	if item.GetType() != "cloudwatch-instance-metric" {
		t.Errorf("expected type cloudwatch-instance-metric, got %s", item.GetType())
	}

	// Verify all metrics are present
	attrMap := item.GetAttributes().GetAttrStruct().AsMap()
	for _, metricName := range ec2InstanceMetrics {
		if _, exists := attrMap[metricName]; !exists {
			t.Errorf("expected metric %s to be present in attributes", metricName)
		}
	}
}

func TestCloudwatchInstanceMetricAdapterGetWrongScope(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	wrongScope := "999999999999.us-east-1"
	query := "i-1234567890abcdef0"

	_, err := adapter.Get(context.Background(), wrongScope, query, false)
	if err == nil {
		t.Error("expected error for wrong scope, got nil")
	}
}

func TestCloudwatchInstanceMetricAdapterGetInvalidQuery(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scope := "123456789012.eu-west-2"

	tests := []struct {
		name  string
		query string
	}{
		{"invalid format", "not-an-instance-id"},
		{"too short", "i-123"},
		{"missing prefix", "1234567890abcdef0"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.Get(context.Background(), scope, tt.query, false)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestCloudwatchInstanceMetricAdapterList(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scope := "123456789012.eu-west-2"

	items, err := adapter.List(context.Background(), scope, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List should return empty - we can't list all instance metrics
	if len(items) != 0 {
		t.Errorf("expected 0 items from List, got %d", len(items))
	}
}

func TestCloudwatchInstanceMetricAdapterScopes(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scopes := adapter.Scopes()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if scopes[0] != "123456789012.eu-west-2" {
		t.Errorf("expected scope 123456789012.eu-west-2, got %s", scopes[0])
	}
}

func TestCloudwatchInstanceMetricAdapterMetadata(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClient{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	metadata := adapter.Metadata()
	if metadata == nil {
		t.Fatal("expected metadata, got nil")
	}
	if metadata.GetType() != "cloudwatch-instance-metric" {
		t.Errorf("expected type cloudwatch-instance-metric, got %s", metadata.GetType())
	}
}

func TestNewCloudwatchInstanceMetricAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := cloudwatch.NewFromConfig(config)

	adapter := NewCloudwatchInstanceMetricAdapter(client, account, region, nil)

	if adapter.Type() != "cloudwatch-instance-metric" {
		t.Errorf("expected type cloudwatch-instance-metric, got %s", adapter.Type())
	}

	if adapter.Name() != "cloudwatch-instance-metric-adapter" {
		t.Errorf("expected name cloudwatch-instance-metric-adapter, got %s", adapter.Name())
	}
}

func TestCloudwatchInstanceMetricAdapterCaching(t *testing.T) {
	ctx := t.Context()
	client := &testCloudwatchMetricClientWithCallCount{}
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    client,
		AccountID: "123456789012",
		Region:    "eu-west-2",
		SDPCache:  sdpcache.NewCache(ctx),
	}

	scope := "123456789012.eu-west-2"
	query := "i-1234567890abcdef0"

	// First call should hit the API
	first, err := adapter.Get(context.Background(), scope, query, false)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if first == nil {
		t.Fatal("expected first item, got nil")
	}
	if client.callCount != 1 {
		t.Errorf("expected 1 API call, got %d", client.callCount)
	}

	// Second call should use cache (ignoreCache=false)
	second, err := adapter.Get(context.Background(), scope, query, false)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if second == nil {
		t.Fatal("expected second item, got nil")
	}
	// Should still be 1 call since we used cache
	if client.callCount != 1 {
		t.Errorf("expected 1 API call after cache hit, got %d", client.callCount)
	}

	// Verify both items are the same (from cache)
	// Compare by checking the InstanceId attribute
	firstAttrs := first.GetAttributes().GetAttrStruct().AsMap()
	secondAttrs := second.GetAttributes().GetAttrStruct().AsMap()
	if firstAttrs["InstanceId"] != secondAttrs["InstanceId"] {
		t.Error("cached item should match original item")
	}
}

func TestCloudwatchInstanceMetricAdapterIgnoreCache(t *testing.T) {
	client := &testCloudwatchMetricClientWithCallCount{}
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    client,
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scope := "123456789012.eu-west-2"
	query := "i-1234567890abcdef0"

	// First call should hit the API
	_, err := adapter.Get(context.Background(), scope, query, false)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if client.callCount != 1 {
		t.Errorf("expected 1 API call, got %d", client.callCount)
	}

	// Second call with ignoreCache=true should bypass cache and hit API again
	_, err = adapter.Get(context.Background(), scope, query, true)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	// Should be 2 calls since we ignored cache
	if client.callCount != 2 {
		t.Errorf("expected 2 API calls after ignoreCache=true, got %d", client.callCount)
	}
}

// testCloudwatchMetricClientError always returns an error
type testCloudwatchMetricClientError struct{}

func (c testCloudwatchMetricClientError) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return nil, fmt.Errorf("API error")
}

func TestCloudwatchInstanceMetricAdapterErrorCaching(t *testing.T) {
	adapter := &CloudwatchInstanceMetricAdapter{
		Client:    testCloudwatchMetricClientError{},
		AccountID: "123456789012",
		Region:    "eu-west-2",
	}

	scope := "123456789012.eu-west-2"
	query := "i-1234567890abcdef0"

	// First call should fail and cache the error
	_, err := adapter.Get(context.Background(), scope, query, false)
	if err == nil {
		t.Fatal("expected error on first call, got nil")
	}

	// Second call should return the cached error without calling the API again
	// We can't easily verify the API wasn't called, but we can verify the same error is returned
	_, err2 := adapter.Get(context.Background(), scope, query, false)
	if err2 == nil {
		t.Fatal("expected cached error on second call, got nil")
	}
	if err.Error() != err2.Error() {
		t.Errorf("expected same error message, got different: %v vs %v", err, err2)
	}
}
