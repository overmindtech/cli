package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestS3SearchImpl(t *testing.T) {
	cache := sdpcache.NewNoOpCache()
	t.Run("with S3 bucket ARN format (empty account ID and region)", func(t *testing.T) {
		// This test verifies that S3 bucket ARNs with empty account ID and region work correctly
		// Format: arn:aws:s3:::bucket-name
		// When parsed, AccountID="", Region="", so FormatScope("", "") returns sdp.WILDCARD
		// The adapter skips scope validation when accountID is empty and uses its own scope
		//
		// EXPECTED BEHAVIOR: Search should succeed because S3 bucket ARNs don't include account/region
		// (S3 is global), and the adapter should use its own scope since it knows the account ID.
		bucketName := "test-bucket-name"
		s3ARN := "arn:aws:s3:::" + bucketName
		adapterScope := "account-id" // S3 scopes are account-only (no region)

		items, err := searchImpl(context.Background(), cache, TestS3Client{}, adapterScope, s3ARN, false)

		// We EXPECT this to succeed, but it currently fails with NOSCOPE error
		// This test demonstrates the bug existing
		if err != nil {
			var ire *sdp.QueryError
			if errors.As(err, &ire) {
				if ire.GetErrorType() == sdp.QueryError_NOSCOPE && strings.Contains(ire.GetErrorString(), "ARN scope") {
					// This is the bug - the search fails when it should succeed
					t.Errorf("BUG REPRODUCED: Search failed with NOSCOPE error when it should succeed. "+
						"Error: %v. S3 bucket ARNs don't include account/region, so the adapter should use its own scope.",
						ire.GetErrorString())
					t.Logf("Expected: Search succeeds and returns bucket item")
					t.Logf("Actual: Search fails with NOSCOPE error: %v", ire.GetErrorString())
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				t.Errorf("unexpected error type: %T: %v", err, err)
			}
			return
		}

		// If we get here, the search succeeded (expected behavior)
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %v", len(items))
		}
		if items[0] == nil {
			t.Error("expected non-nil item")
		}
	})
}

func TestS3ListImpl(t *testing.T) {
	cache := sdpcache.NewNoOpCache()
	items, err := listImpl(context.Background(), cache, TestS3Client{}, "foo", false)

	if err != nil {
		t.Error(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}
}

func TestS3GetImpl(t *testing.T) {
	cache := sdpcache.NewNoOpCache()
	item, err := getImpl(context.Background(), cache, TestS3Client{}, "foo", "bar", false)

	if err != nil {
		t.Fatal(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "https://hostname",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "lambda-function",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "sqs-queue",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "sns-topic",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "bucket",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:s3:::amzn-s3-demo-bucket",
			ExpectedScope:  sdp.WILDCARD,
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:s3:::amzn-s3-demo-bucket",
			ExpectedScope:  sdp.WILDCARD,
		},
	}

	tests.Execute(t, item)
}

func TestS3SourceCaching(t *testing.T) {
	cache := sdpcache.NewMemoryCache()
	first, err := getImpl(context.Background(), cache, TestS3Client{}, "foo", "bar", false)
	if err != nil {
		t.Fatal(err)
	}
	if first == nil {
		t.Fatal("expected first item")
	}

	second, err := getImpl(context.Background(), cache, TestS3FailClient{}, "foo", "bar", false)
	if err != nil {
		t.Fatal(err)
	}
	if second == nil {
		t.Fatal("expected second item")
	}

	third, err := getImpl(context.Background(), cache, TestS3Client{}, "foo", "bar", true)
	if err != nil {
		t.Fatal(err)
	}
	if third == nil {
		t.Fatal("expected third item")
	}

	if third == second {
		t.Errorf("expected third item (%v) to be different to second item (%v)", third, second)
	}
}

var owner = types.Owner{
	DisplayName: PtrString("dylan"),
	ID:          PtrString("id"),
}

// TestS3Client A client that returns example data
type TestS3Client struct{}

func (t TestS3Client) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return &s3.ListBucketsOutput{
		Buckets: []types.Bucket{
			{
				CreationDate: PtrTime(time.Now()),
				Name:         PtrString("foo"),
			},
		},
		Owner: &owner,
	}, nil
}

func (t TestS3Client) GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
	return &s3.GetBucketAclOutput{
		Grants: []types.Grant{
			{
				Grantee: &types.Grantee{
					Type:         types.TypeAmazonCustomerByEmail,
					DisplayName:  PtrString("dylan"),
					EmailAddress: PtrString("dylan@company.com"),
					ID:           PtrString("id"),
					URI:          PtrString("uri"),
				},
			},
		},
		Owner: &owner,
	}, nil
}

func (t TestS3Client) GetBucketAnalyticsConfiguration(ctx context.Context, params *s3.GetBucketAnalyticsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketAnalyticsConfigurationOutput, error) {
	return &s3.GetBucketAnalyticsConfigurationOutput{
		AnalyticsConfiguration: &types.AnalyticsConfiguration{
			Id: PtrString("id"),
			StorageClassAnalysis: &types.StorageClassAnalysis{
				DataExport: &types.StorageClassAnalysisDataExport{
					Destination: &types.AnalyticsExportDestination{
						S3BucketDestination: &types.AnalyticsS3BucketDestination{
							Bucket:          PtrString("arn:aws:s3:::amzn-s3-demo-bucket"),
							Format:          types.AnalyticsS3ExportFileFormatCsv,
							BucketAccountId: PtrString("id"),
							Prefix:          PtrString("pre"),
						},
					},
					OutputSchemaVersion: types.StorageClassAnalysisSchemaVersionV1,
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketCors(ctx context.Context, params *s3.GetBucketCorsInput, optFns ...func(*s3.Options)) (*s3.GetBucketCorsOutput, error) {
	return &s3.GetBucketCorsOutput{
		CORSRules: []types.CORSRule{
			{
				AllowedMethods: []string{
					"GET",
				},
				AllowedOrigins: []string{
					"amazon.com",
				},
				AllowedHeaders: []string{
					"Authorization",
				},
				ExposeHeaders: []string{
					"foo",
				},
				ID:            PtrString("id"),
				MaxAgeSeconds: PtrInt32(10),
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	return &s3.GetBucketEncryptionOutput{
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm:   types.ServerSideEncryptionAes256,
						KMSMasterKeyID: PtrString("id"),
					},
					BucketKeyEnabled: PtrBool(true),
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketIntelligentTieringConfiguration(ctx context.Context, params *s3.GetBucketIntelligentTieringConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketIntelligentTieringConfigurationOutput, error) {
	return &s3.GetBucketIntelligentTieringConfigurationOutput{
		IntelligentTieringConfiguration: &types.IntelligentTieringConfiguration{
			Id:     PtrString("id"),
			Status: types.IntelligentTieringStatusEnabled,
			Tierings: []types.Tiering{
				{
					AccessTier: types.IntelligentTieringAccessTierDeepArchiveAccess,
					Days:       PtrInt32(100),
				},
			},
			Filter: &types.IntelligentTieringFilter{},
		},
	}, nil
}

func (t TestS3Client) GetBucketInventoryConfiguration(ctx context.Context, params *s3.GetBucketInventoryConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketInventoryConfigurationOutput, error) {
	return &s3.GetBucketInventoryConfigurationOutput{
		InventoryConfiguration: &types.InventoryConfiguration{
			Destination: &types.InventoryDestination{
				S3BucketDestination: &types.InventoryS3BucketDestination{
					Bucket:    PtrString("arn:aws:s3:::amzn-s3-demo-bucket"),
					Format:    types.InventoryFormatCsv,
					AccountId: PtrString("id"),
					Encryption: &types.InventoryEncryption{
						SSEKMS: &types.SSEKMS{
							KeyId: PtrString("key"),
						},
					},
					Prefix: PtrString("pre"),
				},
			},
			Id:                     PtrString("id"),
			IncludedObjectVersions: types.InventoryIncludedObjectVersionsAll,
			IsEnabled:              PtrBool(true),
			Schedule: &types.InventorySchedule{
				Frequency: types.InventoryFrequencyDaily,
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketLifecycleConfiguration(ctx context.Context, params *s3.GetBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	return &s3.GetBucketLifecycleConfigurationOutput{
		Rules: []types.LifecycleRule{
			{
				Status: types.ExpirationStatusEnabled,
				AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
					DaysAfterInitiation: PtrInt32(1),
				},
				Expiration: &types.LifecycleExpiration{
					Date:                      PtrTime(time.Now()),
					Days:                      PtrInt32(3),
					ExpiredObjectDeleteMarker: PtrBool(true),
				},
				ID: PtrString("id"),
				NoncurrentVersionExpiration: &types.NoncurrentVersionExpiration{
					NewerNoncurrentVersions: PtrInt32(3),
					NoncurrentDays:          PtrInt32(1),
				},
				NoncurrentVersionTransitions: []types.NoncurrentVersionTransition{
					{
						NewerNoncurrentVersions: PtrInt32(1),
						NoncurrentDays:          PtrInt32(1),
						StorageClass:            types.TransitionStorageClassGlacierIr,
					},
				},
				Prefix: PtrString("pre"),
				Transitions: []types.Transition{
					{
						Date:         PtrTime(time.Now()),
						Days:         PtrInt32(12),
						StorageClass: types.TransitionStorageClassGlacierIr,
					},
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return &s3.GetBucketLocationOutput{
		LocationConstraint: types.BucketLocationConstraintAfSouth1,
	}, nil
}

func (t TestS3Client) GetBucketLogging(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error) {
	return &s3.GetBucketLoggingOutput{
		LoggingEnabled: &types.LoggingEnabled{
			TargetBucket: PtrString("bucket"),
			TargetPrefix: PtrString("pre"),
			TargetGrants: []types.TargetGrant{
				{
					Grantee: &types.Grantee{
						Type: types.TypeGroup,
						ID:   PtrString("id"),
					},
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketMetricsConfiguration(ctx context.Context, params *s3.GetBucketMetricsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketMetricsConfigurationOutput, error) {
	return &s3.GetBucketMetricsConfigurationOutput{
		MetricsConfiguration: &types.MetricsConfiguration{
			Id: PtrString("id"),
		},
	}, nil
}

func (t TestS3Client) GetBucketNotificationConfiguration(ctx context.Context, params *s3.GetBucketNotificationConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketNotificationConfigurationOutput, error) {
	return &s3.GetBucketNotificationConfigurationOutput{
		LambdaFunctionConfigurations: []types.LambdaFunctionConfiguration{
			{
				Events:            []types.Event{},
				LambdaFunctionArn: PtrString("arn:partition:service:region:account-id:resource-type:resource-id"),
				Id:                PtrString("id"),
			},
		},
		EventBridgeConfiguration: &types.EventBridgeConfiguration{},
		QueueConfigurations: []types.QueueConfiguration{
			{
				Events:   []types.Event{},
				QueueArn: PtrString("arn:partition:service:region:account-id:resource-type:resource-id"),
				Filter: &types.NotificationConfigurationFilter{
					Key: &types.S3KeyFilter{
						FilterRules: []types.FilterRule{
							{
								Name:  types.FilterRuleNamePrefix,
								Value: PtrString("foo"),
							},
						},
					},
				},
				Id: PtrString("id"),
			},
		},
		TopicConfigurations: []types.TopicConfiguration{
			{
				Events:   []types.Event{},
				TopicArn: PtrString("arn:partition:service:region:account-id:resource-type:resource-id"),
				Filter: &types.NotificationConfigurationFilter{
					Key: &types.S3KeyFilter{
						FilterRules: []types.FilterRule{
							{
								Name:  types.FilterRuleNameSuffix,
								Value: PtrString("fix"),
							},
						},
					},
				},
				Id: PtrString("id"),
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketOwnershipControls(ctx context.Context, params *s3.GetBucketOwnershipControlsInput, optFns ...func(*s3.Options)) (*s3.GetBucketOwnershipControlsOutput, error) {
	return &s3.GetBucketOwnershipControlsOutput{
		OwnershipControls: &types.OwnershipControls{
			Rules: []types.OwnershipControlsRule{
				{
					ObjectOwnership: types.ObjectOwnershipBucketOwnerPreferred,
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
	return &s3.GetBucketPolicyOutput{
		Policy: PtrString("policy"),
	}, nil
}

func (t TestS3Client) GetBucketPolicyStatus(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error) {
	return &s3.GetBucketPolicyStatusOutput{
		PolicyStatus: &types.PolicyStatus{
			IsPublic: PtrBool(true),
		},
	}, nil
}

func (t TestS3Client) GetBucketReplication(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error) {
	return &s3.GetBucketReplicationOutput{
		ReplicationConfiguration: &types.ReplicationConfiguration{
			Role: PtrString("role"),
			Rules: []types.ReplicationRule{
				{
					Destination: &types.Destination{
						Bucket: PtrString("bucket"),
						AccessControlTranslation: &types.AccessControlTranslation{
							Owner: types.OwnerOverrideDestination,
						},
						Account: PtrString("account"),
						EncryptionConfiguration: &types.EncryptionConfiguration{
							ReplicaKmsKeyID: PtrString("keyId"),
						},
						Metrics: &types.Metrics{
							Status: types.MetricsStatusEnabled,
							EventThreshold: &types.ReplicationTimeValue{
								Minutes: PtrInt32(1),
							},
						},
						ReplicationTime: &types.ReplicationTime{
							Status: types.ReplicationTimeStatusEnabled,
							Time: &types.ReplicationTimeValue{
								Minutes: PtrInt32(1),
							},
						},
						StorageClass: types.StorageClassGlacier,
					},
				},
			},
		},
	}, nil
}

func (t TestS3Client) GetBucketRequestPayment(ctx context.Context, params *s3.GetBucketRequestPaymentInput, optFns ...func(*s3.Options)) (*s3.GetBucketRequestPaymentOutput, error) {
	return &s3.GetBucketRequestPaymentOutput{
		Payer: types.PayerRequester,
	}, nil
}

func (t TestS3Client) GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return &s3.GetBucketTaggingOutput{
		TagSet: []types.Tag{},
	}, nil
}

func (t TestS3Client) GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	return &s3.GetBucketVersioningOutput{
		MFADelete: types.MFADeleteStatusEnabled,
		Status:    types.BucketVersioningStatusSuspended,
	}, nil
}

func (t TestS3Client) GetBucketWebsite(ctx context.Context, params *s3.GetBucketWebsiteInput, optFns ...func(*s3.Options)) (*s3.GetBucketWebsiteOutput, error) {
	return &s3.GetBucketWebsiteOutput{
		ErrorDocument: &types.ErrorDocument{
			Key: PtrString("key"),
		},
		IndexDocument: &types.IndexDocument{
			Suffix: PtrString("html"),
		},
		RedirectAllRequestsTo: &types.RedirectAllRequestsTo{
			HostName: PtrString("hostname"),
			Protocol: types.ProtocolHttps,
		},
		RoutingRules: []types.RoutingRule{
			{
				Redirect: &types.Redirect{
					HostName:             PtrString("hostname"),
					HttpRedirectCode:     PtrString("303"),
					Protocol:             types.ProtocolHttp,
					ReplaceKeyPrefixWith: PtrString("pre"),
					ReplaceKeyWith:       PtrString("key"),
				},
			},
		},
	}, nil
}

type TestS3FailClient struct{}

func (t TestS3FailClient) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return nil, errors.New("failed to list buckets")
}

func (t TestS3FailClient) GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
	return nil, errors.New("failed to get bucket ACL")
}
func (t TestS3FailClient) GetBucketAnalyticsConfiguration(ctx context.Context, params *s3.GetBucketAnalyticsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketAnalyticsConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket ACL")
}

func (t TestS3FailClient) GetBucketCors(ctx context.Context, params *s3.GetBucketCorsInput, optFns ...func(*s3.Options)) (*s3.GetBucketCorsOutput, error) {
	return nil, errors.New("failed to get bucket CORS")
}

func (t TestS3FailClient) GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	return nil, errors.New("failed to get bucket CORS")
}

func (t TestS3FailClient) GetBucketIntelligentTieringConfiguration(ctx context.Context, params *s3.GetBucketIntelligentTieringConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketIntelligentTieringConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket CORS")
}

func (t TestS3FailClient) GetBucketInventoryConfiguration(ctx context.Context, params *s3.GetBucketInventoryConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketInventoryConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket CORS")
}

func (t TestS3FailClient) GetBucketLifecycleConfiguration(ctx context.Context, params *s3.GetBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket lifecycle configuration")
}

func (t TestS3FailClient) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return nil, errors.New("failed to get bucket location")
}

func (t TestS3FailClient) GetBucketLogging(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error) {
	return nil, errors.New("failed to get bucket logging")
}

func (t TestS3FailClient) GetBucketMetricsConfiguration(ctx context.Context, params *s3.GetBucketMetricsConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketMetricsConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket logging")
}

func (t TestS3FailClient) GetBucketNotificationConfiguration(ctx context.Context, params *s3.GetBucketNotificationConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketNotificationConfigurationOutput, error) {
	return nil, errors.New("failed to get bucket notification configuration")
}

func (t TestS3FailClient) GetBucketOwnershipControls(ctx context.Context, params *s3.GetBucketOwnershipControlsInput, optFns ...func(*s3.Options)) (*s3.GetBucketOwnershipControlsOutput, error) {
	return nil, errors.New("failed to get bucket policy")
}

func (t TestS3FailClient) GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
	return nil, errors.New("failed to get bucket policy")
}

func (t TestS3FailClient) GetBucketPolicyStatus(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error) {
	return nil, errors.New("failed to get bucket policy")
}

func (t TestS3FailClient) GetBucketReplication(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error) {
	return nil, errors.New("failed to get bucket replication")
}

func (t TestS3FailClient) GetBucketRequestPayment(ctx context.Context, params *s3.GetBucketRequestPaymentInput, optFns ...func(*s3.Options)) (*s3.GetBucketRequestPaymentOutput, error) {
	return nil, errors.New("failed to get bucket request payment")
}

func (t TestS3FailClient) GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return nil, errors.New("failed to get bucket tagging")
}

func (t TestS3FailClient) GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	return nil, errors.New("failed to get bucket versioning")
}

func (t TestS3FailClient) GetBucketWebsite(ctx context.Context, params *s3.GetBucketWebsiteInput, optFns ...func(*s3.Options)) (*s3.GetBucketWebsiteOutput, error) {
	return nil, errors.New("failed to get bucket website")
}

func (t TestS3FailClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("failed to get object")
}

func (t TestS3FailClient) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return nil, errors.New("failed to head bucket")
}

func (t TestS3FailClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("failed to head object")
}

func (t TestS3FailClient) PutBucketAcl(ctx context.Context, params *s3.PutBucketAclInput, optFns ...func(*s3.Options)) (*s3.PutBucketAclOutput, error) {
	return nil, errors.New("failed to put bucket ACL")
}

func (t TestS3FailClient) PutBucketCors(ctx context.Context, params *s3.PutBucketCorsInput, optFns ...func(*s3.Options)) (*s3.PutBucketCorsOutput, error) {
	return nil, errors.New("failed to put bucket CORS")
}

func (t TestS3FailClient) PutBucketLifecycleConfiguration(ctx context.Context, params *s3.PutBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return nil, errors.New("failed to put bucket lifecycle configuration")
}

func (t TestS3FailClient) PutBucketLogging(ctx context.Context, params *s3.PutBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketLoggingOutput, error) {
	return nil, errors.New("failed to put bucket logging")
}

func (t TestS3FailClient) PutBucketNotificationConfiguration(ctx context.Context, params *s3.PutBucketNotificationConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketNotificationConfigurationOutput, error) {
	return nil, errors.New("failed to put bucket notification configuration")
}

func (t TestS3FailClient) PutBucketPolicy(ctx context.Context, params *s3.PutBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error) {
	return nil, errors.New("failed to put bucket policy")
}

func (t TestS3FailClient) PutBucketReplication(ctx context.Context, params *s3.PutBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.PutBucketReplicationOutput, error) {
	return nil, errors.New("failed to put bucket replication")
}

func (t TestS3FailClient) PutBucketRequestPayment(ctx context.Context, params *s3.PutBucketRequestPaymentInput, optFns ...func(*s3.Options)) (*s3.PutBucketRequestPaymentOutput, error) {
	return nil, errors.New("failed to put bucket request payment")
}

func (t TestS3FailClient) PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error) {
	return nil, errors.New("failed to put bucket tagging")
}

func (t TestS3FailClient) PutBucketVersioning(ctx context.Context, params *s3.PutBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	return nil, errors.New("failed to put bucket versioning")
}

func (t TestS3FailClient) PutBucketWebsite(ctx context.Context, params *s3.PutBucketWebsiteInput, optFns ...func(*s3.Options)) (*s3.PutBucketWebsiteOutput, error) {
	return nil, errors.New("failed to put bucket website")
}

func (t TestS3FailClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return nil, errors.New("failed to put object")
}

func TestNewS3Adapter(t *testing.T) {
	config, account, _ := GetAutoConfig(t)

	adapter := NewS3Adapter(config, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}

func TestS3SearchWithARNFormat(t *testing.T) {
	// This E2E test reproduces the customer issue:
	// - Get works with bucket name: harness-sample-three-qa-us-west-2-20251022151048279100000001
	// - Search fails with ARN: arn:aws:s3:::harness-sample-three-qa-us-west-2-20251022151048279100000001
	//
	// EXPECTED BEHAVIOR: Both Get and Search should work
	// CURRENT BEHAVIOR: Get works, Search fails with NOSCOPE error - THIS IS THE BUG
	config, account, _ := GetAutoConfig(t)

	adapter := NewS3Adapter(config, account, sdpcache.NewNoOpCache())
	scope := adapter.Scopes()[0]

	bucketName := "harness-sample-three-qa-us-west-2-20251022151048279100000001"
	s3ARN := "arn:aws:s3:::" + bucketName

	ctx := context.Background()

	// First, verify that Get works with the bucket name directly
	t.Run("Get with bucket name", func(t *testing.T) {
		item, err := adapter.Get(ctx, scope, bucketName, false)
		if err != nil {
			t.Logf("Get failed (this is OK if bucket doesn't exist): %v", err)
		} else if item != nil {
			t.Logf("Get succeeded: found bucket %v", bucketName)
		}
	})

	// Then, test Search with ARN format - this SHOULD succeed, but currently fails with NOSCOPE error
	t.Run("Search with S3 ARN format", func(t *testing.T) {
		items, err := adapter.Search(ctx, scope, s3ARN, false)

		// EXPECTED: Search succeeds because S3 bucket ARNs don't include account/region
		// (S3 is global), and the adapter should use its own scope since it knows the account ID.
		// CURRENT: Search fails with NOSCOPE error - THIS IS THE BUG
		if err != nil {
			var ire *sdp.QueryError
			if errors.As(err, &ire) {
				if ire.GetErrorType() == sdp.QueryError_NOSCOPE && strings.Contains(ire.GetErrorString(), "ARN scope") {
					// This is the bug - the search fails when it should succeed
					t.Errorf("BUG REPRODUCED: Search failed with NOSCOPE error when it should succeed. "+
						"Error: %v. S3 bucket ARNs don't include account/region, so the adapter should use its own scope.",
						ire.GetErrorString())
					t.Logf("Expected: Search succeeds and returns bucket item (like Get does)")
					t.Logf("Actual: Search fails with NOSCOPE error: %v", ire.GetErrorString())
				} else {
					// Other errors (like bucket not found) are acceptable
					t.Logf("Search failed with error (may be expected if bucket doesn't exist): %v", err)
				}
			} else {
				t.Errorf("unexpected error type: %T: %v", err, err)
			}
			return
		}

		// If we get here, the search succeeded (expected behavior)
		if len(items) == 0 {
			t.Error("expected at least 1 item from Search")
		} else {
			t.Logf("Search succeeded: found %v item(s)", len(items))
		}
	})
}
