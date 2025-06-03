package auth

import (
	"context"

	"github.com/posthog/posthog-go"
	log "github.com/sirupsen/logrus"
)

func FeatureFlagEnabled(ctx context.Context, posthogClient posthog.Client, key string) (enabled bool) {
	// get the account name from the context
	accountName, ok := ctx.Value(AccountNameContextKey{}).(string)
	if !ok || accountName == "" {
		log.WithContext(ctx).WithFields(log.Fields{"ovm.auth.accountName": accountName, "key": key}).Warn("account name is not set in context, cannot check feature flag")
		return false
	}
	// get the subject from the context
	subject, ok := ctx.Value(CurrentSubjectContextKey{}).(string)
	if !ok || subject == "" {
		log.WithContext(ctx).WithFields(log.Fields{"ovm.auth.subject": subject, "ovm.auth.accountName": accountName, "key": key}).Warn("subject is not set in context, cannot check feature flag")
		return false
	}

	return FeatureFlagEnabledCustom(ctx, posthogClient, accountName, subject, key)
}

// FeatureFlagEnabledCustom checks if a feature flag is enabled for a given subject and account.
// Retries are handled by the posthog client internally. by default
// Creates a `DefaultBacko` instance with the following defaults: https://github.com/segmentio/backo-go/blob/master/backo.go
//
//	base: 100 milliseconds
//	factor: 2
//	jitter: 0
//	cap: 10 seconds
//
// http://posthog.com/docs/libraries/go#feature-flags
func FeatureFlagEnabledCustom(ctx context.Context, posthogClient posthog.Client, accountName, subject, key string) (enabled bool) {
	lf := log.Fields{"ovm.auth.subject": subject, "ovm.auth.accountName": accountName, "key": key}
	if posthogClient == nil {
		log.WithContext(ctx).WithFields(lf).Warn("posthog client is nil, cannot check feature flag")
		return false
	}
	properties := posthog.NewProperties()

	properties.Set("accountName", accountName)
	err := posthogClient.Enqueue(posthog.Identify{
		DistinctId: subject,
		Properties: properties,
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Warn("could not enqueue identify event in posthog")
		return false
	}

	rawFeatureEnabled, err := posthogClient.IsFeatureEnabled(posthog.FeatureFlagPayload{
		Key:              key,
		DistinctId:       subject,
		PersonProperties: properties,
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Warn("could not check feature flag in posthog")
		return false
	}
	log.WithContext(ctx).WithFields(lf).Debugf("feature flag %s for subject %s is %t", key, subject, rawFeatureEnabled)
	return rawFeatureEnabled == true
}
