package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsHttp "github.com/aws/smithy-go/transport/http"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// FormatScope Formats an account ID and region into the corresponding Overmind
// scope. This will be in the format {accountID}.{region}
func FormatScope(accountID, region string) string {
	if region == "" {
		return accountID
	}

	return fmt.Sprintf("%v.%v", accountID, region)
}

// ParseScope Parses a scope and returns the account id and region
func ParseScope(scope string) (string, string, error) {
	sections := strings.Split(scope, ".")

	if len(sections) != 2 {
		return "", "", fmt.Errorf("could not split scope '%v' into 2 sections", scope)
	}

	return sections[0], sections[1], nil
}

// Returns whether or not it makes sense to retry the error. This can be used to
// decide whether we should cache the error or not. Errors such as the item
// being not found, or the scope not existing should not be retried for example
func CanRetry(err *sdp.QueryError) bool {
	switch err.GetErrorType() { //nolint:exhaustive
	case sdp.QueryError_NOTFOUND, sdp.QueryError_NOSCOPE:
		return false
	default:
		return true
	}
}

// A parsed representation of the parts of the ARN that Overmind needs to care
// about
//
// Format example:
//
//	arn:partition:service:region:account-id:resource-type:resource-id
type ARN struct {
	arn.ARN
}

// ResourceID The ID of the resource, this is everything after the type and
// might also include a version or other components depending on the service
// e.g. ecs-template-ecs-demo-app:1 would be the ResourceID for
// "arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"
func (a *ARN) ResourceID() string {
	// Find the first separator
	separatorLocation := strings.IndexFunc(a.Resource, func(r rune) bool {
		return r == '/' || r == ':'
	})

	// Remove the first field since this is the type, then keep the rest
	return a.Resource[separatorLocation+1:]
}

// Type The type of the resource, this is everything after the service and
// before the resource ID
//
// e.g. "task-definition" would be the Type for
// "arn:aws:ecs:eu-west-1:052392120703:task-definition/ecs-template-ecs-demo-app:1"
func (a *ARN) Type() string {
	// Find the first separator
	separatorLocation := strings.IndexFunc(a.Resource, func(r rune) bool {
		return r == '/' || r == ':'
	})

	if separatorLocation == -1 {
		return a.Resource
	}

	// Keep the first field since this is the type, then remove the rest
	return a.Resource[:separatorLocation]
}

// Matches checks if the IAM wildcards included in the ARN match another ARN
// using the logic that IAM uses. For example if the ARN is
// "arn:aws:s3:::amzn-s3-demo-bucket/*" then it will match
// "arn:aws:s3:::amzn-s3-demo-bucket/thing" but not
// "arn:aws:s3:::some-other-bucket/object"
func (a *ARN) IAMWildcardMatches(arn string) bool {
	targetARN, err := ParseARN(arn)
	if err != nil {
		return false
	}

	// You can't use a wildcard in the service segment
	if a.Service != targetARN.Service {
		return false
	}

	// Convert * wildcard to regex pattern and escape other special chars
	convertToPattern := func(s string) string {
		// Escape regex special chars except * and ?
		special := []string{".", "+", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
		escaped := s
		for _, ch := range special {
			escaped = strings.ReplaceAll(escaped, ch, "\\"+ch)
		}
		// Convert * to .* and ? to . for regex
		escaped = strings.ReplaceAll(escaped, "*", ".*")
		escaped = strings.ReplaceAll(escaped, "?", ".")
		return "^" + escaped + "$"
	}

	// Check each component using pattern matching
	components := []struct {
		pattern string
		target  string
	}{
		{a.Region, targetARN.Region},
		{a.AccountID, targetARN.AccountID},
		{a.Resource, targetARN.Resource},
	}

	for _, c := range components {
		pattern := convertToPattern(c.pattern)
		matched, err := regexp.MatchString(pattern, c.target)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

func (a *ARN) ContainsWildcard() bool {
	possibleWildcardLocations := a.Partition + a.Region + a.AccountID + a.Resource
	return strings.Contains(possibleWildcardLocations, "*") || strings.Contains(possibleWildcardLocations, "?")
}

// ParseARN Parses an ARN and tries to determine the resource ID from it. The
// logic is that the resource ID will be the last component when separated by
// slashes or colons: https://devopscube.com/aws-arn-guide/
func ParseARN(arnString string) (*ARN, error) {
	a, err := arn.Parse(arnString)
	if err != nil {
		return nil, err
	}

	return &ARN{
		ARN: a,
	}, nil
}

// WrapAWSError Wraps an AWS error in the appropriate SDP error
func WrapAWSError(err error) *sdp.QueryError {
	var responseErr *awsHttp.ResponseError

	if errors.As(err, &responseErr) {
		// If the input is bad, access is denied, or the thing wasn't found then
		// we should assume that it is not exist for this adapter
		if slices.Contains([]int{400, 403, 404}, responseErr.HTTPStatusCode()) {
			return &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: err.Error(),
			}
		}
	}

	return sdp.NewQueryError(err)
}

// Adds an event to the span to note the error, and returns a set of tags that
// return a standardised set of tags that contains `errorGettingTags` and
// `error`
func HandleTagsError(ctx context.Context, err error) map[string]string {
	if err == nil {
		return nil
	}

	// Attach an event in the span
	span := trace.SpanFromContext(ctx)

	span.AddEvent("Error getting tags", trace.WithAttributes(
		attribute.String("error", err.Error()),
	))

	return map[string]string{
		"errorGettingTags": "true",
		"error":            err.Error(),
	}
}

// E2ETest A struct that runs end to end tests on a fully configured adapters.
// These tests aren't particularly detailed, but they are designed to ensure
// that there aren't any really obvious error when it's actually configured with
// AWS credentials
type E2ETest struct {
	// The adapter to test
	Adapter discovery.Adapter

	// A search query that should return > 0 results
	GoodSearchQuery *string

	// Skips get tests
	SkipGet bool

	// Skips list tests
	SkipList bool

	// Skips checking that a know bad get query returns a NOTFOUND error
	SkipNotFoundCheck bool

	// A timeout used for all tests
	Timeout time.Duration
}

// The purpose of these tests is mostly to give an entrypoint for debugging in a
// real environment
func (e E2ETest) Run(t *testing.T) {
	t.Parallel()
	t.Helper()

	type Validator interface {
		Validate() error
	}

	if v, ok := e.Adapter.(Validator); ok {
		if err := v.Validate(); err != nil {
			t.Fatalf("adapter failed validation: %v", err)
		}
	}

	// Determine the scope so that we can use this for all queries
	scopes := e.Adapter.Scopes()
	if len(scopes) == 0 {
		t.Fatalf("some scopes, got %v", len(scopes))
	}
	scope := scopes[0]

	t.Run(fmt.Sprintf("Adapter: %v", e.Adapter.Name()), func(t *testing.T) {
		if e.GoodSearchQuery != nil {
			t.Run(fmt.Sprintf("Good search query: %v", e.GoodSearchQuery), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
				defer cancel()

				var items []*sdp.Item
				var err error
				if searchSrc, ok := e.Adapter.(discovery.SearchableAdapter); ok {
					items, err = searchSrc.Search(ctx, scope, *e.GoodSearchQuery, false)
				} else if streamSrc, ok := e.Adapter.(discovery.SearchStreamableAdapter); ok {
					stream := discovery.NewRecordingQueryResultStream()
					streamSrc.SearchStream(context.Background(), scope, *e.GoodSearchQuery, false, stream)

					if len(stream.GetErrors()) > 0 {
						err = stream.GetErrors()[0]
					}

					items = stream.GetItems()
				} else {
					t.Skip("adapter is not searchable or streamable")
				}

				if err != nil {
					t.Error(err)
				}

				if len(items) == 0 {
					t.Error("no items returned")
				}

				for _, item := range items {
					if err = item.Validate(); err != nil {
						t.Error(err)
					}

					if item.GetType() != e.Adapter.Type() {
						t.Errorf("mismatched item type \"%v\" and adapter type \"%v\"", item.GetType(), e.Adapter.Type())
					}
				}
			})
		}

		t.Run("List query", func(t *testing.T) {
			if e.SkipList {
				t.Skip("list tests deliberately skipped")
			}

			var items []*sdp.Item
			errs := make([]error, 0)

			ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
			defer cancel()

			if streamingAdapter, ok := e.Adapter.(discovery.ListStreamableAdapter); ok {
				stream := discovery.NewRecordingQueryResultStream()
				streamingAdapter.ListStream(context.Background(), scope, false, stream)
				items = stream.GetItems()
				errs = stream.GetErrors()
			} else if listableAdapter, ok := e.Adapter.(discovery.ListableAdapter); ok {
				var err error
				items, err = listableAdapter.List(ctx, scope, false)
				if err != nil {
					errs = append(errs, err)
				}
			} else {
				t.Skip("adapter is not listable or streamable")
			}

			allNames := make(map[string]bool)

			for _, err := range errs {
				t.Error(err)
			}

			for _, item := range items {
				if _, exists := allNames[item.UniqueAttributeValue()]; exists {
					t.Errorf("duplicate item found: %v", item.UniqueAttributeValue())
				} else {
					allNames[item.UniqueAttributeValue()] = true
				}

				if err := item.Validate(); err != nil {
					t.Error(err)
				}

				if item.GetType() != e.Adapter.Type() {
					t.Errorf("mismatched item type \"%v\" and adapter type \"%v\"", item.GetType(), e.Adapter.Type())
				}
			}

			if len(items) > 0 {
				// Do a get for a known good item
				query := items[0].UniqueAttributeValue()

				t.Run(fmt.Sprintf("Good get query: %v", query), func(t *testing.T) {
					if e.SkipGet {
						t.Skip("get tests deliberately skipped")
					}

					ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
					defer cancel()

					item, err := e.Adapter.Get(ctx, scope, query, false)
					if err != nil {
						t.Fatal(err)
					}

					if err = item.Validate(); err != nil {
						t.Fatal(err)
					}

					if item.GetType() != e.Adapter.Type() {
						t.Errorf("mismatched item type \"%v\" and adapter type \"%v\"", item.GetType(), e.Adapter.Type())
					}
				})
			}
		})

		t.Run("bad get query", func(t *testing.T) {
			if e.SkipGet {
				t.Skip("get tests deliberately skipped")
			}

			ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
			defer cancel()

			_, err := e.Adapter.Get(ctx, scope, "this is a known bad get query", false)

			if err == nil {
				t.Error("expected error, got nil")
			}

			if !e.SkipNotFoundCheck {
				// Make sure the error is an SDP error
				var sdpErr *sdp.QueryError
				if errors.As(err, &sdpErr) {
					if sdpErr.GetErrorType() != sdp.QueryError_NOTFOUND {
						t.Errorf("expected error to be NOTFOUND, got %v\nError: %v", sdpErr.GetErrorType().String(), sdpErr.GetErrorString())
					}
				} else {
					t.Errorf("Error (%T) was not (*sdp.QueryError)", err)
				}
			}
		})
	})
}

// GetAutoConfig Uses automatic local config (i.e. `aws configure`) to get an
// AWS config object, AWS account ID and region. Skips the tests if this is
// unavailable
func GetAutoConfig(t *testing.T) (aws.Config, string, string) {
	t.Helper()

	config, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		rawCIString := os.Getenv("CI")
		if strings.EqualFold(rawCIString, "true") {
			// These tests were always just really simple smoke tests that relied on data being already populated in AWS.
			// They were just a good way to check the shape of the data coming back during development.
			t.Skip("Skipping test because no AWS credentials are available in CI environment. They are for during development ONLY.")
		} else {
			t.Fatalf("Failed to load default config: %v", err)
		}
	}

	// Add OTel instrumentation
	config.HTTPClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	stsClient := sts.NewFromConfig(config)

	var callerID *sts.GetCallerIdentityOutput

	callerID, err = stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("Failed to get caller identity, for config: %+v. %v", config, err)
	}

	return config, *callerID.Account, config.Region
}

// Converts an interface to SDP attributes using the `sdp.ToAttributesSorted`
// function, and also allows the user to exclude certain top-level fields from
// the resulting attributes
func ToAttributesWithExclude(i interface{}, exclusions ...string) (*sdp.ItemAttributes, error) {
	attrs, err := sdp.ToAttributesViaJson(i)
	if err != nil {
		return nil, err
	}

	for _, exclusion := range exclusions {
		if s := attrs.GetAttrStruct(); s != nil {
			delete(s.GetFields(), exclusion)
		}
	}

	return attrs, nil
}
