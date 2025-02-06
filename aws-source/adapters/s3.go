package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/getsentry/sentry-go"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

const CacheDuration = 10 * time.Minute

// NewS3Source Creates a new S3 adapter
func NewS3Adapter(config aws.Config, accountID string) *S3Source {
	return &S3Source{
		config:          config,
		accountID:       accountID,
		AdapterMetadata: s3Metadata,
	}
}

var s3Metadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "s3-bucket",
	DescriptiveName: "S3 Bucket",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an S3 bucket by name",
		ListDescription:   "List all S3 buckets",
		SearchDescription: "Search for S3 buckets by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_s3_bucket_acl.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_analytics_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_cors_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_intelligent_tiering_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_inventory.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_lifecycle_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_logging.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_metric.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_notification.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_object_lock_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_object.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_ownership_controls.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_policy.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_public_access_block.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_replication_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_request_payment_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_server_side_encryption_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_versioning.bucket"},
		{TerraformQueryMap: "aws_s3_bucket_website_configuration.bucket"},
		{TerraformQueryMap: "aws_s3_bucket.id"},
		{TerraformQueryMap: "aws_s3_object_copy.bucket"},
		{TerraformQueryMap: "aws_s3_object.bucket"},
	},
	PotentialLinks: []string{"lambda-function", "sqs-queue", "sns-topic", "s3-bucket"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})

type S3Source struct {
	// AWS Config including region and credentials
	config aws.Config

	// AccountID The id of the account that is being used. This is used by
	// sources as the first element in the scope
	accountID string

	// client The AWS client to use when making requests
	client          *s3.Client
	clientCreated   bool
	clientMutex     sync.Mutex
	AdapterMetadata *sdp.AdapterMetadata

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once
}

func (s *S3Source) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *S3Source) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

func (s *S3Source) Client() *s3.Client {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	// If the client already exists then return it
	if s.clientCreated {
		return s.client
	}

	// Otherwise create a new client from the config
	s.client = s3.NewFromConfig(s.config)
	s.clientCreated = true

	return s.client
}

// Type The type of items that this adapter is capable of finding
func (s *S3Source) Type() string {

	return "s3-bucket"
}

// Descriptive name for the adapter, used in logging and metadata
func (s *S3Source) Name() string {
	return "aws-s3-adapter"
}

func (s *S3Source) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

// List of scopes that this adapter is capable of find items for. This will be
// in the format {accountID} since S3 endpoint is global
func (s *S3Source) Scopes() []string {
	return []string{
		adapterhelpers.FormatScope(s.accountID, ""),
	}
}

// S3Client A client that can get data about S3 buckets
type S3Client interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error)
	GetBucketAnalyticsConfiguration(ctx context.Context, params *s3.GetBucketAnalyticsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketAnalyticsConfigurationOutput, error)
	GetBucketCors(ctx context.Context, params *s3.GetBucketCorsInput, optFns ...func(*s3.Options)) (*s3.GetBucketCorsOutput, error)
	GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	GetBucketIntelligentTieringConfiguration(ctx context.Context, params *s3.GetBucketIntelligentTieringConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketIntelligentTieringConfigurationOutput, error)
	GetBucketInventoryConfiguration(ctx context.Context, params *s3.GetBucketInventoryConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketInventoryConfigurationOutput, error)
	GetBucketLifecycleConfiguration(ctx context.Context, params *s3.GetBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLifecycleConfigurationOutput, error)
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	GetBucketLogging(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error)
	GetBucketMetricsConfiguration(ctx context.Context, params *s3.GetBucketMetricsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketMetricsConfigurationOutput, error)
	GetBucketNotificationConfiguration(ctx context.Context, params *s3.GetBucketNotificationConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketNotificationConfigurationOutput, error)
	GetBucketOwnershipControls(ctx context.Context, params *s3.GetBucketOwnershipControlsInput, optFns ...func(*s3.Options)) (*s3.GetBucketOwnershipControlsOutput, error)
	GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error)
	GetBucketPolicyStatus(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error)
	GetBucketReplication(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error)
	GetBucketRequestPayment(ctx context.Context, params *s3.GetBucketRequestPaymentInput, optFns ...func(*s3.Options)) (*s3.GetBucketRequestPaymentOutput, error)
	GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
	GetBucketWebsite(ctx context.Context, params *s3.GetBucketWebsiteInput, optFns ...func(*s3.Options)) (*s3.GetBucketWebsiteOutput, error)
}

// Bucket represents an actual s3 bucket, with all of the extra requests
// resolved and all information added
type Bucket struct {
	// ListBuckets
	types.Bucket

	s3.GetBucketAclOutput
	s3.GetBucketAnalyticsConfigurationOutput
	s3.GetBucketCorsOutput
	s3.GetBucketEncryptionOutput
	s3.GetBucketIntelligentTieringConfigurationOutput
	s3.GetBucketInventoryConfigurationOutput
	s3.GetBucketLifecycleConfigurationOutput
	s3.GetBucketLocationOutput
	s3.GetBucketLoggingOutput
	s3.GetBucketMetricsConfigurationOutput
	s3.GetBucketNotificationConfigurationOutput
	s3.GetBucketOwnershipControlsOutput
	s3.GetBucketPolicyOutput
	s3.GetBucketPolicyStatusOutput
	s3.GetBucketReplicationOutput
	s3.GetBucketRequestPaymentOutput
	s3.GetBucketVersioningOutput
	s3.GetBucketWebsiteOutput
}

// Get Get a single item with a given scope and query. The item returned
// should have a UniqueAttributeValue that matches the `query` parameter. The
// ctx parameter contains a golang context object which should be used to allow
// this adapter to timeout or be cancelled when executing potentially
// long-running actions
func (s *S3Source) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != s.Scopes()[0] {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
			Scope:       scope,
		}
	}

	s.ensureCache()
	return getImpl(ctx, s.cache, s.Client(), scope, query, ignoreCache)
}

func getImpl(ctx context.Context, cache *sdpcache.Cache, client S3Client, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	cacheHit, ck, cachedItems, qErr := cache.Lookup(ctx, "aws-s3-adapter", sdp.QueryMethod_GET, scope, "s3-bucket", query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		} else {
			return nil, nil
		}
	}

	var location *s3.GetBucketLocationOutput
	var wg sync.WaitGroup
	var err error

	bucketName := adapterhelpers.PtrString(query)

	location, err = client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: bucketName,
	})

	if err != nil {
		err = adapterhelpers.WrapAWSError(err)
		cache.StoreError(err, CacheDuration, ck)
		return nil, err
	}

	bucket := Bucket{
		Bucket: types.Bucket{
			Name: bucketName,
		},
		GetBucketLocationOutput: *location,
	}

	// We want to execute all of these requests in parallel so we're not
	// crippled by latency. This API is really stupid but there's not much I can
	// do about it
	var tagging *s3.GetBucketTaggingOutput

	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if acl, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketAclOutput = *acl
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if analyticsConfiguration, err := client.GetBucketAnalyticsConfiguration(ctx, &s3.GetBucketAnalyticsConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketAnalyticsConfigurationOutput = *analyticsConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if cors, err := client.GetBucketCors(ctx, &s3.GetBucketCorsInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketCorsOutput = *cors
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if encryption, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketEncryptionOutput = *encryption
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if intelligentTieringConfiguration, err := client.GetBucketIntelligentTieringConfiguration(ctx, &s3.GetBucketIntelligentTieringConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketIntelligentTieringConfigurationOutput = *intelligentTieringConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if inventoryConfiguration, err := client.GetBucketInventoryConfiguration(ctx, &s3.GetBucketInventoryConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketInventoryConfigurationOutput = *inventoryConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if lifecycleConfiguration, err := client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketLifecycleConfigurationOutput = *lifecycleConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if logging, err := client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketLoggingOutput = *logging
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if metricsConfiguration, err := client.GetBucketMetricsConfiguration(ctx, &s3.GetBucketMetricsConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketMetricsConfigurationOutput = *metricsConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if notificationConfiguration, err := client.GetBucketNotificationConfiguration(ctx, &s3.GetBucketNotificationConfigurationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketNotificationConfigurationOutput = *notificationConfiguration
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if ownershipControls, err := client.GetBucketOwnershipControls(ctx, &s3.GetBucketOwnershipControlsInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketOwnershipControlsOutput = *ownershipControls
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if policy, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketPolicyOutput = *policy
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if policyStatus, err := client.GetBucketPolicyStatus(ctx, &s3.GetBucketPolicyStatusInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketPolicyStatusOutput = *policyStatus
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if replication, err := client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketReplicationOutput = *replication
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if requestPayment, err := client.GetBucketRequestPayment(ctx, &s3.GetBucketRequestPaymentInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketRequestPaymentOutput = *requestPayment
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if out, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: bucketName}); err == nil {
			tagging = out
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if versioning, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketVersioningOutput = *versioning
		}
	}()
	wg.Add(1)
	go func() {
		defer sentry.Recover()
		defer wg.Done()
		if website, err := client.GetBucketWebsite(ctx, &s3.GetBucketWebsiteInput{Bucket: bucketName}); err == nil {
			bucket.GetBucketWebsiteOutput = *website
		}
	}()

	// Wait for all requests to complete
	wg.Wait()

	attributes, err := adapterhelpers.ToAttributesWithExclude(bucket)

	if err != nil {
		err = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
		cache.StoreError(err, CacheDuration, ck)
		return nil, err
	}

	// Convert tags
	tags := make(map[string]string)

	if tagging != nil {
		for _, tag := range tagging.TagSet {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	item := sdp.Item{
		Type:            "s3-bucket",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            tags,
	}

	if bucket.RedirectAllRequestsTo != nil {
		if bucket.RedirectAllRequestsTo.HostName != nil {
			var url string

			switch bucket.RedirectAllRequestsTo.Protocol {
			case types.ProtocolHttp:
				url = "https://" + *bucket.RedirectAllRequestsTo.HostName
			case types.ProtocolHttps:
				url = "https://" + *bucket.RedirectAllRequestsTo.HostName
			}

			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  url,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// HTTP always linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	var a *adapterhelpers.ARN

	for _, lambdaConfig := range bucket.LambdaFunctionConfigurations {
		if lambdaConfig.LambdaFunctionArn != nil {
			if a, err = adapterhelpers.ParseARN(*lambdaConfig.LambdaFunctionArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "lambda-function",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *lambdaConfig.LambdaFunctionArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	for _, q := range bucket.QueueConfigurations {
		if q.QueueArn != nil {
			if a, err = adapterhelpers.ParseARN(*q.QueueArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "sqs-queue",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *q.QueueArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	for _, topic := range bucket.TopicConfigurations {
		if topic.TopicArn != nil {
			if a, err = adapterhelpers.ParseARN(*topic.TopicArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "sns-topic",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *topic.TopicArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	if bucket.LoggingEnabled != nil {
		if bucket.LoggingEnabled.TargetBucket != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "s3-bucket",
					Method: sdp.QueryMethod_GET,
					Query:  *bucket.LoggingEnabled.TargetBucket,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled
					In:  true,
					Out: true,
				},
			})
		}
	}

	if bucket.InventoryConfiguration != nil {
		if bucket.InventoryConfiguration.Destination != nil {
			if bucket.InventoryConfiguration.Destination.S3BucketDestination != nil {
				if bucket.InventoryConfiguration.Destination.S3BucketDestination.Bucket != nil {
					if a, err = adapterhelpers.ParseARN(*bucket.InventoryConfiguration.Destination.S3BucketDestination.Bucket); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "s3-bucket",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *bucket.InventoryConfiguration.Destination.S3BucketDestination.Bucket,
								Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
							},
							BlastPropagation: &sdp.BlastPropagation{
								// Tightly coupled
								In:  true,
								Out: true,
							},
						})
					}
				}
			}
		}
	}

	// Dear god there has to be a better way to do this? Should we just let it
	// panic and then deal with it?
	if bucket.AnalyticsConfiguration != nil {
		if bucket.AnalyticsConfiguration.StorageClassAnalysis != nil {
			if bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport != nil {
				if bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport.Destination != nil {
					if bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport.Destination.S3BucketDestination != nil {
						if bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport.Destination.S3BucketDestination.Bucket != nil {
							if a, err = adapterhelpers.ParseARN(*bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport.Destination.S3BucketDestination.Bucket); err == nil {
								item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
									Query: &sdp.Query{
										Type:   "s3-bucket",
										Method: sdp.QueryMethod_SEARCH,
										Query:  *bucket.AnalyticsConfiguration.StorageClassAnalysis.DataExport.Destination.S3BucketDestination.Bucket,
										Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
									},
									BlastPropagation: &sdp.BlastPropagation{
										// Tightly coupled
										In:  true,
										Out: true,
									},
								})
							}
						}
					}
				}
			}
		}
	}

	cache.StoreItem(&item, CacheDuration, ck)

	return &item, nil
}

// List Lists all items in a given scope
func (s *S3Source) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != s.Scopes()[0] {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
			Scope:       scope,
		}
	}

	s.ensureCache()
	return listImpl(ctx, s.cache, s.Client(), scope, ignoreCache)
}

func listImpl(ctx context.Context, cache *sdpcache.Cache, client S3Client, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	cacheHit, ck, cachedItems, qErr := cache.Lookup(ctx, "aws-s3-adapter", sdp.QueryMethod_LIST, scope, "s3-bucket", "", ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		} else {
			return nil, nil
		}
	}

	items := make([]*sdp.Item, 0)

	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})

	if err != nil {
		err = sdp.NewQueryError(err)
		cache.StoreError(err, CacheDuration, ck)
		return nil, err
	}

	for _, bucket := range buckets.Buckets {
		item, err := getImpl(ctx, cache, client, scope, *bucket.Name, ignoreCache)

		if err != nil {
			continue
		}

		items = append(items, item)
	}

	for _, item := range items {
		cache.StoreItem(item, CacheDuration, ck)
	}
	return items, nil
}

// Search Searches for an S3 bucket by ARN rather than name
func (s *S3Source) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != s.Scopes()[0] {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
			Scope:       scope,
		}
	}

	s.ensureCache()
	return searchImpl(ctx, s.cache, s.Client(), scope, query, ignoreCache)
}

func searchImpl(ctx context.Context, cache *sdpcache.Cache, client S3Client, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	// Parse the ARN
	a, err := adapterhelpers.ParseARN(query)

	if err != nil {
		return nil, sdp.NewQueryError(err)
	}

	if arnScope := adapterhelpers.FormatScope(a.AccountID, a.Region); arnScope != scope {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("ARN scope %v does not match adapters scope %v", arnScope, scope),
			Scope:       scope,
		}
	}

	// If the ARN was parsed we can just ask Get for the item
	item, err := getImpl(ctx, cache, client, scope, a.ResourceID(), ignoreCache)
	if err != nil {
		return nil, err
	}

	return []*sdp.Item{item}, nil
}

// Weight Returns the priority weighting of items returned by this adapter.
// This is used to resolve conflicts where two sources of the same type
// return an item for a GET request. In this instance only one item can be
// seen on, so the one with the higher weight value will win.
func (s *S3Source) Weight() int {
	return 100
}
