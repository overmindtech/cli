package shared

import (
	"context"
	"errors"

	"cloud.google.com/go/storage"
)

// ErrStorageClientNotInitialized is returned when the Storage client was not initialized (e.g. when enumerating adapters without initGCPClients).
var ErrStorageClientNotInitialized = errors.New("storage client not initialized")

// BucketIAMBinding represents one IAM binding (role + members, optionally with a condition) in a bucket's policy.
// The adapter emits one item per bucket (the full policy); bindings are serialized in that item's bindings array.
type BucketIAMBinding struct {
	Role                 string
	Members              []string
	ConditionExpression  string // CEL expression; empty if no condition
	ConditionTitle       string // optional; empty if no condition or not set
	ConditionDescription string // optional; empty if no condition or not set
}

// StorageBucketIAMPolicyGetter retrieves the IAM policy for a GCS bucket as a slice of bindings.
// See: https://cloud.google.com/storage/docs/json_api/v1/buckets/getIamPolicy
type StorageBucketIAMPolicyGetter interface {
	GetBucketIAMPolicy(ctx context.Context, bucketName string) ([]BucketIAMBinding, error)
}

// storageBucketIAMPolicyGetterImpl implements StorageBucketIAMPolicyGetter using the Storage client.
type storageBucketIAMPolicyGetterImpl struct {
	client *storage.Client
}

// GetBucketIAMPolicy returns the IAM policy for the given bucket.
func (g *storageBucketIAMPolicyGetterImpl) GetBucketIAMPolicy(ctx context.Context, bucketName string) ([]BucketIAMBinding, error) {
	if g.client == nil {
		return nil, ErrStorageClientNotInitialized
	}
	policy3, err := g.client.Bucket(bucketName).IAM().V3().Policy(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BucketIAMBinding, 0, len(policy3.Bindings))
	for _, b := range policy3.Bindings {
		condExpr := ""
		condTitle := ""
		condDesc := ""
		if b.GetCondition() != nil {
			condExpr = b.GetCondition().GetExpression()
			condTitle = b.GetCondition().GetTitle()
			condDesc = b.GetCondition().GetDescription()
		}
		out = append(out, BucketIAMBinding{
			Role:                 b.GetRole(),
			Members:              b.GetMembers(),
			ConditionExpression:  condExpr,
			ConditionTitle:       condTitle,
			ConditionDescription: condDesc,
		})
	}
	return out, nil
}

// NewStorageBucketIAMPolicyGetter creates a getter that uses the given Storage client.
func NewStorageBucketIAMPolicyGetter(client *storage.Client) StorageBucketIAMPolicyGetter {
	return &storageBucketIAMPolicyGetterImpl{client: client}
}
