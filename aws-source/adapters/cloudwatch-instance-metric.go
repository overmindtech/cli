package adapters

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

// CloudwatchMetricClient defines the CloudWatch client interface for metrics
type CloudwatchMetricClient interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

// EC2 instance metrics to fetch
// Metric units (as returned by CloudWatch with Average statistic over 15-minute period):
// - CPUUtilization: Percentage (0-100)
// - NetworkIn: Average bytes per second
// - NetworkOut: Average bytes per second
// - StatusCheckFailed: Count (0 = OK, 1 = Failed)
// - CPUCreditBalance: Number of CPU credits available (for T2/T3 instances)
// - CPUCreditUsage: Number of CPU credits consumed (for T2/T3 instances)
// - DiskReadOps: Average read operations per second (instance store volumes)
// - DiskWriteOps: Average write operations per second (instance store volumes)
var ec2InstanceMetrics = []string{
	"CPUUtilization",
	"NetworkIn",
	"NetworkOut",
	"StatusCheckFailed",
	"CPUCreditBalance",
	"CPUCreditUsage",
	"DiskReadOps",
	"DiskWriteOps",
}

// validateInstanceID validates that the query is a valid EC2 instance ID
func validateInstanceID(instanceID string) error {
	// EC2 instance IDs start with "i-" followed by either 8 characters (older instances)
	// or 17 characters (newer instances, default since 2016). Both use hexadecimal characters (0-9, a-f).
	matched, err := regexp.MatchString(`^i-[0-9a-f]{8}$|^i-[0-9a-f]{17}$`, instanceID)
	if err != nil {
		return fmt.Errorf("failed to validate instance ID: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid instance ID format: %s (expected format: i-xxxxxxxx or i-xxxxxxxxxxxxxxxxx)", instanceID)
	}
	return nil
}

// formatBytes formats bytes to human-readable format (KB, MB, GB, TB)
func formatBytes(bytes float64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", bytes/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", bytes/KB)
	default:
		return fmt.Sprintf("%.0f bytes", bytes)
	}
}

// formatBytesPerSecond formats bytes per second to human-readable format
func formatBytesPerSecond(bytesPerSec float64) string {
	return formatBytes(bytesPerSec) + "/s"
}

// formatOpsPerSecond formats operations per second
func formatOpsPerSecond(opsPerSec float64) string {
	if opsPerSec >= 1000000 {
		return fmt.Sprintf("%.2f M ops/s", opsPerSec/1000000)
	}
	if opsPerSec >= 1000 {
		return fmt.Sprintf("%.2f K ops/s", opsPerSec/1000)
	}
	return fmt.Sprintf("%.2f ops/s", opsPerSec)
}

// formatMetricValue formats a metric value based on its name
func formatMetricValue(metricName string, value float64) string {
	switch metricName {
	case "CPUUtilization":
		return fmt.Sprintf("%.2f%%", value)
	case "NetworkIn", "NetworkOut":
		// These are average bytes per second over the 15-minute period
		return formatBytesPerSecond(value)
	case "StatusCheckFailed":
		// This is a count (0 or 1), show as boolean-like
		if value == 0 {
			return "OK"
		}
		return "Failed"
	case "CPUCreditBalance", "CPUCreditUsage":
		// These are counts of credits
		return fmt.Sprintf("%.2f credits", value)
	case "DiskReadOps", "DiskWriteOps":
		// These are average operations per second over the 15-minute period
		return formatOpsPerSecond(value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// metricOutputMapper converts CloudWatch GetMetricData output to an SDP item
func metricOutputMapper(ctx context.Context, client CloudwatchMetricClient, scope string, instanceID string, output *cloudwatch.GetMetricDataOutput) (*sdp.Item, error) {
	// Build attributes map with instance ID
	attrsMap := map[string]interface{}{
		"InstanceId":    instanceID,
		"PeriodMinutes": 15,
		"Statistic":     "Average",
		"DataAvailable": false,
		"LastUpdated":   "",
	}

	// Map metric results to attributes
	var lastTime time.Time
	hasData := false

	for _, result := range output.MetricDataResults {
		if len(result.Values) > 0 && len(result.Timestamps) > 0 {
			// Get the most recent value (last in the arrays)
			value := result.Values[len(result.Values)-1]
			timestamp := result.Timestamps[len(result.Timestamps)-1]

			// Use the metric label as the attribute name
			metricName := aws.ToString(result.Label)

			// Store raw value
			attrsMap[metricName] = value

			// Store formatted value for human readability
			formattedKey := metricName + "_Formatted"
			attrsMap[formattedKey] = formatMetricValue(metricName, value)

			// Track the most recent timestamp
			if timestamp.After(lastTime) {
				lastTime = timestamp
			}
			hasData = true
		}
	}

	attrsMap["DataAvailable"] = hasData
	if !lastTime.IsZero() {
		attrsMap["LastUpdated"] = lastTime.Format(time.RFC3339)
	}

	attrs, err := sdp.ToAttributes(attrsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attributes: %w", err)
	}

	item := &sdp.Item{
		Type:            "cloudwatch-instance-metric",
		UniqueAttribute: "InstanceId",
		Scope:           scope,
		Attributes:      attrs,
	}

	return item, nil
}

// CloudwatchInstanceMetricAdapter is a custom adapter for CloudWatch EC2 instance metrics
type CloudwatchInstanceMetricAdapter struct {
	Client        CloudwatchMetricClient
	AccountID     string
	Region        string
	CacheDuration time.Duration  // How long to cache items for
	cache         sdpcache.Cache // The cache for this adapter (set during creation, can be nil for tests)
}

// Default cache duration for metrics - matches the 15-minute period over which metrics are averaged
const defaultMetricCacheDuration = 15 * time.Minute

func (a *CloudwatchInstanceMetricAdapter) cacheDuration() time.Duration {
	if a.CacheDuration == 0 {
		return defaultMetricCacheDuration
	}
	return a.CacheDuration
}

var (
	noOpCacheCloudwatchOnce sync.Once
	noOpCacheCloudwatch     sdpcache.Cache
)

func (a *CloudwatchInstanceMetricAdapter) Cache() sdpcache.Cache {
	if a.cache == nil {
		noOpCacheCloudwatchOnce.Do(func() {
			noOpCacheCloudwatch = sdpcache.NewNoOpCache()
		})
		return noOpCacheCloudwatch
	}
	return a.cache
}

// Type returns the type of items this adapter returns
func (a *CloudwatchInstanceMetricAdapter) Type() string {
	return "cloudwatch-instance-metric"
}

// Name returns the name of this adapter
func (a *CloudwatchInstanceMetricAdapter) Name() string {
	return "cloudwatch-instance-metric-adapter"
}

// Metadata returns the adapter metadata
func (a *CloudwatchInstanceMetricAdapter) Metadata() *sdp.AdapterMetadata {
	return cloudwatchInstanceMetricAdapterMetadata
}

// Scopes returns the scopes this adapter can query
func (a *CloudwatchInstanceMetricAdapter) Scopes() []string {
	return []string{FormatScope(a.AccountID, a.Region)}
}

// Get fetches CloudWatch metrics for an EC2 instance by instance ID
func (a *CloudwatchInstanceMetricAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != FormatScope(a.AccountID, a.Region) {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("scope %s does not match adapter scope %s", scope, FormatScope(a.AccountID, a.Region)),
			Scope:       scope,
		}
	}

	// Query is just the instance ID
	instanceID := query
	if err := validateInstanceID(instanceID); err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
	}

	// Check cache first
	var cacheHit bool
	var ck sdpcache.CacheKey
	var cachedItems []*sdp.Item
	var qErr *sdp.QueryError

	cacheHit, ck, cachedItems, qErr, done := a.Cache().Lookup(ctx, a.Name(), sdp.QueryMethod_GET, scope, a.Type(), query, ignoreCache)
	defer done()
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit && len(cachedItems) > 0 {
		return cachedItems[0], nil
	}

	// Query CloudWatch for the last 15 minutes
	endTime := time.Now()
	startTime := endTime.Add(-15 * time.Minute)

	// Build metric data queries for all metrics
	metricQueries := make([]types.MetricDataQuery, 0, len(ec2InstanceMetrics))
	for i, metricName := range ec2InstanceMetrics {
		id := fmt.Sprintf("m%d", i)
		metricQueries = append(metricQueries, types.MetricDataQuery{
			Id: aws.String(id),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String("AWS/EC2"),
					MetricName: aws.String(metricName),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("InstanceId"),
							Value: aws.String(instanceID),
						},
					},
				},
				Period: aws.Int32(900), // 15 minutes
				Stat:   aws.String("Average"),
			},
			Label: aws.String(metricName),
		})
	}

	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: metricQueries,
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
	}

	output, err := a.Client.GetMetricData(ctx, input)
	if err != nil {
		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to get metric data: %v", err),
			Scope:       scope,
		}
		// Cache the error
		a.Cache().StoreError(ctx, qErr, a.cacheDuration(), ck)
		return nil, qErr
	}

	item, err := metricOutputMapper(ctx, a.Client, scope, instanceID, output)
	if err != nil {
		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to map metric output: %v", err),
			Scope:       scope,
		}
		// Cache the error
		a.Cache().StoreError(ctx, qErr, a.cacheDuration(), ck)
		return nil, qErr
	}

	// Store in cache
	a.Cache().StoreItem(ctx, item, a.cacheDuration(), ck)
	return item, nil
}

// List is not supported for instance metrics - you must query specific instances
func (a *CloudwatchInstanceMetricAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	// Listing all instance metrics is not practical
	// Return empty list with no error
	return []*sdp.Item{}, nil
}

// Search is not supported for instance metrics
func (a *CloudwatchInstanceMetricAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	// Search delegates to Get for this adapter
	item, err := a.Get(ctx, scope, query, ignoreCache)
	if err != nil {
		return nil, err
	}
	return []*sdp.Item{item}, nil
}

// Weight returns the priority weight of this adapter
func (a *CloudwatchInstanceMetricAdapter) Weight() int {
	return 100
}

// NewCloudwatchInstanceMetricAdapter creates a new CloudWatch instance metric adapter
func NewCloudwatchInstanceMetricAdapter(client *cloudwatch.Client, accountID string, region string, cache sdpcache.Cache) *CloudwatchInstanceMetricAdapter {
	return &CloudwatchInstanceMetricAdapter{
		Client:    client,
		AccountID: accountID,
		Region:    region,
		cache:  cache,
	}
}

var cloudwatchInstanceMetricAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudwatch-instance-metric",
	DescriptiveName: "CloudWatch Instance Metric",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              false, // Listing all instance metrics is not practical
		Search:            true,
		GetDescription:    "Get CloudWatch metrics for an EC2 instance by instance ID (e.g., 'i-1234567890abcdef0')",
		SearchDescription: "Search for CloudWatch metrics for an EC2 instance using instance ID (e.g., 'i-1234567890abcdef0')",
	},
	PotentialLinks: []string{"ec2-instance"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
