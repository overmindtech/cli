package discovery

import (
	"context"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
)

type SlowAdapter struct {
	QueryDuration time.Duration
}

func (s *SlowAdapter) Type() string {
	return "person"
}

func (s *SlowAdapter) Name() string {
	return "slow-adapter"
}

func (s *SlowAdapter) DefaultCacheDuration() time.Duration {
	return 10 * time.Minute
}

func (s *SlowAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{}
}

func (s *SlowAdapter) Scopes() []string {
	return []string{"test"}
}

func (s *SlowAdapter) Hidden() bool {
	return false
}

func (s *SlowAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	end := time.Now().Add(s.QueryDuration)
	attributes, _ := sdp.ToAttributes(map[string]interface{}{
		"name": query,
	})

	item := sdp.Item{
		Type:            "person",
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           "test",
		// TODO(LIQs): delete this
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// TODO(LIQs): convert to returning edges
	for i := 0; i != 2; i++ {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  RandomName(),
			Scope:  "test",
		}})
	}

	time.Sleep(time.Until(end))

	return &item, nil
}

func (s *SlowAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return []*sdp.Item{}, nil
}

func (s *SlowAdapter) Weight() int {
	return 100
}

func TestParallelQueryPerformance(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Performance tests under github actions are too unreliable")
	}

	// This test is designed to ensure that query duration is linear up to a
	// certain point. Above that point the overhead caused by having so many
	// goroutines running will start to make the response times non-linear which
	// maybe isn't ideal but given realistic loads we probably don't care.
	t.Run("Without linking", func(t *testing.T) {
		RunLinearPerformanceTest(t, "1 query", 1, 0, 1)
		RunLinearPerformanceTest(t, "10 queries", 10, 0, 1)
		RunLinearPerformanceTest(t, "100 queries", 100, 0, 10)
		RunLinearPerformanceTest(t, "1,000 queries", 1000, 0, 100)
	})
}

// RunLinearPerformanceTest Runs a test with a given number in input queries,
// link depth and parallelization limit. Expected results and expected duration
// are determined automatically meaning all this is testing for is the fact that
// the performance continues to be linear and predictable
func RunLinearPerformanceTest(t *testing.T, name string, numQueries int, linkDepth int, numParallel int) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		result := TimeQueries(t, numQueries, linkDepth, numParallel)

		if len(result.Results) != result.ExpectedItems {
			t.Errorf("Expected %v items, got %v (%v errors)", result.ExpectedItems, len(result.Results), len(result.Errors))
		}

		if result.TimeTaken > result.MaxTime {
			t.Errorf("Queries took too long: %v Max: %v", result.TimeTaken.String(), result.MaxTime.String())
		}
	})
}

type TimedResults struct {
	ExpectedItems int
	MaxTime       time.Duration
	TimeTaken     time.Duration
	Results       []*sdp.Item
	Errors        []*sdp.QueryError
}

func TimeQueries(t *testing.T, numQueries int, linkDepth int, numParallel int) TimedResults {
	ec := EngineConfig{
		MaxParallelExecutions: numParallel,
		NATSOptions: &auth.NATSOptions{
			NumRetries:        5,
			RetryDelay:        time.Second,
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}
	err = e.AddAdapters(&SlowAdapter{
		QueryDuration: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Error adding adapter: %v", err)
	}
	err = e.Start()
	if err != nil {
		t.Fatalf("Error starting Engine: %v", err)
	}
	defer func() {
		err = e.Stop()
		if err != nil {
			t.Fatalf("Error stopping Engine: %v", err)
		}
	}()

	// Calculate how many items to expect and the expected duration
	var expectedItems int
	var expectedDuration time.Duration
	for i := 0; i <= linkDepth; i++ {
		thisLayer := int(math.Pow(2, float64(i))) * numQueries

		// Expect that it'll take no longer that 120% of the sleep time.
		thisDuration := 120 * math.Ceil(float64(thisLayer)/float64(numParallel))
		expectedDuration = expectedDuration + (time.Duration(thisDuration) * time.Millisecond)
		expectedItems = expectedItems + thisLayer
	}

	results := make([]*sdp.Item, 0)
	errors := make([]*sdp.QueryError, 0)
	resultsMutex := sync.Mutex{}
	wg := sync.WaitGroup{}

	start := time.Now()

	for range numQueries {
		qt := QueryTracker{
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  RandomName(),
				Scope:  "test",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: uint32(linkDepth),
				},
			},
			Engine: e,
		}

		wg.Add(1)

		go func(qt *QueryTracker) {
			defer wg.Done()

			items, _, errs, _ := qt.Execute(context.Background())

			resultsMutex.Lock()
			results = append(results, items...)
			errors = append(errors, errs...)
			resultsMutex.Unlock()
		}(&qt)
	}

	wg.Wait()

	return TimedResults{
		ExpectedItems: expectedItems,
		MaxTime:       expectedDuration,
		TimeTaken:     time.Since(start),
		Results:       results,
		Errors:        errors,
	}
}
