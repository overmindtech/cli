package discovery

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/sourcegraph/conc/pool"
)

// BenchmarkAddAdapters_GCPScenario simulates the real-world GCP organization scenario
// where we have many projects, regions, and zones creating thousands of adapters
func BenchmarkAddAdapters_GCPScenario(b *testing.B) {
	scenarios := []struct {
		name         string
		projects     int
		regions      int
		zones        int
		adapterTypes int // Simplified: different adapter types per scope level
	}{
		{"Small_5proj", 5, 5, 10, 20},
		{"Medium_23proj", 23, 35, 135, 88},      // Current failing scenario
		{"Large_100proj", 100, 35, 135, 88},     // Enterprise scenario
		{"VeryLarge_500proj", 500, 35, 135, 88}, // Large enterprise
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				adapters := generateGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.adapterTypes)
				sh := NewAdapterHost()
				b.StartTimer()

				start := time.Now()
				err := sh.AddAdapters(adapters...)
				elapsed := time.Since(start)

				b.StopTimer()
				if err != nil {
					b.Fatalf("Failed to add adapters: %v", err)
				}

				totalAdapters := len(adapters)
				b.ReportMetric(float64(totalAdapters), "adapters")
				b.ReportMetric(elapsed.Seconds(), "seconds")
				b.ReportMetric(float64(totalAdapters)/elapsed.Seconds(), "adapters/sec")
			}
		})
	}
}

// BenchmarkAddAdapters_Scaling tests at different scales to demonstrate O(n²) behavior
func BenchmarkAddAdapters_Scaling(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000, 10000, 25000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				adapters := generateSimpleAdapters(size)
				sh := NewAdapterHost()
				b.StartTimer()

				start := time.Now()
				err := sh.AddAdapters(adapters...)
				elapsed := time.Since(start)

				b.StopTimer()
				if err != nil {
					b.Fatalf("Failed to add adapters: %v", err)
				}

				b.ReportMetric(elapsed.Seconds(), "seconds")
				b.ReportMetric(float64(size)/elapsed.Seconds(), "adapters/sec")
			}
		})
	}
}

// BenchmarkAddAdapters_IncrementalAdd simulates adding adapters one project at a time
// This is closer to how it might be used in practice
func BenchmarkAddAdapters_IncrementalAdd(b *testing.B) {
	projects := 100
	regionsPerProject := 35
	zonesPerProject := 135
	typesPerScope := 30

	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		sh := NewAdapterHost()
		b.StartTimer()

		start := time.Now()

		// Add adapters project by project (like we do in the real code)
		for p := range projects {
			projectAdapters := generateProjectAdapters(p, regionsPerProject, zonesPerProject, typesPerScope)
			err := sh.AddAdapters(projectAdapters...)
			if err != nil {
				b.Fatalf("Failed to add adapters for project %d: %v", p, err)
			}
		}

		elapsed := time.Since(start)
		b.StopTimer()

		totalAdapters := len(sh.Adapters())
		b.ReportMetric(float64(totalAdapters), "total_adapters")
		b.ReportMetric(elapsed.Seconds(), "seconds")
		b.ReportMetric(float64(totalAdapters)/elapsed.Seconds(), "adapters/sec")
	}
}

// generateGCPLikeAdapters creates adapters that mimic the GCP source structure:
// - Project-level adapters (one per project per type)
// - Regional adapters (one per project per region per type)
// - Zonal adapters (one per project per zone per type)
func generateGCPLikeAdapters(projects, regions, zones, typesPerScope int) []Adapter {
	projectTypes := typesPerScope / 3
	regionalTypes := typesPerScope / 3
	zonalTypes := typesPerScope / 3

	totalAdapters := (projects * projectTypes) +
		(projects * regions * regionalTypes) +
		(projects * zones * zonalTypes)

	adapters := make([]Adapter, 0, totalAdapters)

	for p := range projects {
		projectID := fmt.Sprintf("project-%d", p)

		// Project-level adapters
		for t := range projectTypes {
			adapters = append(adapters, &TestAdapter{
				ReturnScopes: []string{projectID},
				ReturnType:   fmt.Sprintf("gcp-project-type-%d", t),
				ReturnName:   fmt.Sprintf("adapter-%s-type-%d", projectID, t),
			})
		}

		// Regional adapters
		for r := range regions {
			scope := fmt.Sprintf("%s.region-%d", projectID, r)
			for t := range regionalTypes {
				adapters = append(adapters, &TestAdapter{
					ReturnScopes: []string{scope},
					ReturnType:   fmt.Sprintf("gcp-regional-type-%d", t),
					ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
				})
			}
		}

		// Zonal adapters
		for z := range zones {
			scope := fmt.Sprintf("%s.zone-%d", projectID, z)
			for t := range zonalTypes {
				adapters = append(adapters, &TestAdapter{
					ReturnScopes: []string{scope},
					ReturnType:   fmt.Sprintf("gcp-zonal-type-%d", t),
					ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
				})
			}
		}
	}

	return adapters
}

// generateProjectAdapters creates all adapters for a single project
func generateProjectAdapters(projectNum, regions, zones, typesPerScope int) []Adapter {
	projectTypes := typesPerScope / 3
	regionalTypes := typesPerScope / 3
	zonalTypes := typesPerScope / 3

	totalAdapters := projectTypes + (regions * regionalTypes) + (zones * zonalTypes)
	adapters := make([]Adapter, 0, totalAdapters)

	projectID := fmt.Sprintf("project-%d", projectNum)

	// Project-level adapters
	for t := range projectTypes {
		adapters = append(adapters, &TestAdapter{
			ReturnScopes: []string{projectID},
			ReturnType:   fmt.Sprintf("gcp-project-type-%d", t),
			ReturnName:   fmt.Sprintf("adapter-%s-type-%d", projectID, t),
		})
	}

	// Regional adapters
	for r := range regions {
		scope := fmt.Sprintf("%s.region-%d", projectID, r)
		for t := range regionalTypes {
			adapters = append(adapters, &TestAdapter{
				ReturnScopes: []string{scope},
				ReturnType:   fmt.Sprintf("gcp-regional-type-%d", t),
				ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
			})
		}
	}

	// Zonal adapters
	for z := range zones {
		scope := fmt.Sprintf("%s.zone-%d", projectID, z)
		for t := range zonalTypes {
			adapters = append(adapters, &TestAdapter{
				ReturnScopes: []string{scope},
				ReturnType:   fmt.Sprintf("gcp-zonal-type-%d", t),
				ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
			})
		}
	}

	return adapters
}

// generateSimpleAdapters creates n unique adapters for simple scaling tests
func generateSimpleAdapters(n int) []Adapter {
	adapters := make([]Adapter, 0, n)
	for i := range n {
		adapters = append(adapters, &TestAdapter{
			ReturnScopes: []string{fmt.Sprintf("scope-%d", i)},
			ReturnType:   fmt.Sprintf("type-%d", i%100), // Reuse 100 types
			ReturnName:   fmt.Sprintf("adapter-%d", i),
		})
	}
	return adapters
}

// BenchmarkListAdapter is a test adapter that returns 10 items per LIST query
// instead of the default 1 item. This is used for memory benchmarks to simulate
// realistic query execution patterns.
type BenchmarkListAdapter struct {
	TestAdapter
	itemsPerList int // Number of items to return per LIST query
}

// List returns exactly 10 items (or itemsPerList if set) for each LIST query
func (b *BenchmarkListAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	// Use the embedded TestAdapter's List method logic but return multiple items
	// We'll call the parent's cache lookup, but then generate multiple items
	itemsPerList := b.itemsPerList
	if itemsPerList == 0 {
		itemsPerList = 10 // Default to 10 items
	}

	cacheHit, ck, cachedItems, qErr, done := b.cache.Lookup(ctx, b.Name(), sdp.QueryMethod_LIST, scope, b.Type(), "", ignoreCache)
	defer done()
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		// If we have cached items, return them (they should already be 10 items from previous call)
		return cachedItems, nil
	}

	// Track the call
	b.ListCalls = append(b.ListCalls, []string{scope})

	switch scope {
	case "empty":
		err := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no items found",
			Scope:       scope,
		}
		b.cache.StoreError(ctx, err, b.DefaultCacheDuration(), ck)
		return nil, err
	case "error":
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Error for testing",
			Scope:       scope,
		}
	default:
		// Generate exactly itemsPerList items
		items := make([]*sdp.Item, 0, itemsPerList)
		for i := range itemsPerList {
			item := b.NewTestItem(scope, fmt.Sprintf("item-%d", i))
			items = append(items, item)
			b.cache.StoreItem(ctx, item, b.DefaultCacheDuration(), ck)
		}
		return items, nil
	}
}

// generateBenchmarkGCPLikeAdapters creates adapters that mimic the GCP source structure
// but use BenchmarkListAdapter which returns 10 items per LIST query
func generateBenchmarkGCPLikeAdapters(projects, regions, zones, typesPerScope int) []Adapter {
	projectTypes := typesPerScope / 3
	regionalTypes := typesPerScope / 3
	zonalTypes := typesPerScope / 3

	totalAdapters := (projects * projectTypes) +
		(projects * regions * regionalTypes) +
		(projects * zones * zonalTypes)

	adapters := make([]Adapter, 0, totalAdapters)

	for p := range projects {
		projectID := fmt.Sprintf("project-%d", p)

		// Project-level adapters
		for t := range projectTypes {
			adapters = append(adapters, &BenchmarkListAdapter{
				TestAdapter: TestAdapter{
					ReturnScopes: []string{projectID},
					ReturnType:   fmt.Sprintf("gcp-project-type-%d", t),
					ReturnName:   fmt.Sprintf("adapter-%s-type-%d", projectID, t),
				},
				itemsPerList: 10,
			})
		}

		// Regional adapters
		for r := range regions {
			scope := fmt.Sprintf("%s.region-%d", projectID, r)
			for t := range regionalTypes {
				adapters = append(adapters, &BenchmarkListAdapter{
					TestAdapter: TestAdapter{
						ReturnScopes: []string{scope},
						ReturnType:   fmt.Sprintf("gcp-regional-type-%d", t),
						ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
					},
					itemsPerList: 10,
				})
			}
		}

		// Zonal adapters
		for z := range zones {
			scope := fmt.Sprintf("%s.zone-%d", projectID, z)
			for t := range zonalTypes {
				adapters = append(adapters, &BenchmarkListAdapter{
					TestAdapter: TestAdapter{
						ReturnScopes: []string{scope},
						ReturnType:   fmt.Sprintf("gcp-zonal-type-%d", t),
						ReturnName:   fmt.Sprintf("adapter-%s-type-%d", scope, t),
					},
					itemsPerList: 10,
				})
			}
		}
	}

	return adapters
}

// newBenchmarkEngine creates an Engine for benchmarks without requiring NATS connection
// The execution pools are manually initialized so queries can be executed without Start()
func newBenchmarkEngine(adapters ...Adapter) (*Engine, error) {
	ec := &EngineConfig{
		MaxParallelExecutions: 2000,
		SourceName:            "benchmark-engine",
		NATSQueueName:         "",
		Unauthenticated:       true,
		// No NATSOptions - we don't need NATS for benchmarks
	}

	e, err := NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error creating engine: %w", err)
	}

	// Manually initialize execution pools (normally done in Start())
	// This allows us to use ExecuteQuery without connecting to NATS
	e.listExecutionPool = pool.New().WithMaxGoroutines(ec.MaxParallelExecutions)
	e.getExecutionPool = pool.New().WithMaxGoroutines(ec.MaxParallelExecutions)

	if err := e.AddAdapters(adapters...); err != nil {
		return nil, fmt.Errorf("error adding adapters: %w", err)
	}

	return e, nil
}

// TestAddAdapters_LargeScale is a regular test (not benchmark) that validates
// the system can handle a realistic large-scale scenario
func TestAddAdapters_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale test in short mode")
	}

	scenarios := []struct {
		name     string
		projects int
		regions  int
		zones    int
		types    int
		timeout  time.Duration
	}{
		{"23_projects", 23, 35, 135, 88, 30 * time.Second},
		{"100_projects", 100, 35, 135, 88, 5 * time.Minute},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			adapters := generateGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.types)
			sh := NewAdapterHost()

			t.Logf("Testing with %d adapters", len(adapters))

			done := make(chan error, 1)
			go func() {
				done <- sh.AddAdapters(adapters...)
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("Failed to add adapters: %v", err)
				}
				t.Logf("Successfully added %d adapters", len(sh.Adapters()))
			case <-time.After(sc.timeout):
				t.Fatalf("AddAdapters timed out after %v (likely O(n²) issue)", sc.timeout)
			}
		})
	}
}

// TestMemoryFootprint_EnterpriseScale measures actual memory usage at enterprise scale
// This provides accurate memory consumption data for capacity planning
func TestMemoryFootprint_EnterpriseScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory footprint test in short mode")
	}

	scenarios := []struct {
		name     string
		projects int
		regions  int
		zones    int
		types    int
	}{
		{"23_projects", 23, 35, 135, 88},
		{"100_projects", 100, 35, 135, 88},
		{"500_projects", 500, 35, 135, 88},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Force GC and get baseline
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			// Create adapters
			adapters := generateGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.types)
			sh := NewAdapterHost()
			err := sh.AddAdapters(adapters...)
			if err != nil {
				t.Fatal(err)
			}

			// Get memory stats immediately (don't GC, we want to see actual usage)
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			// Calculate memory used - use TotalAlloc which is monotonically increasing
			totalAllocated := m2.TotalAlloc - m1.TotalAlloc
			currentHeap := m2.HeapAlloc
			memUsedMB := float64(totalAllocated) / (1024 * 1024)
			heapUsedMB := float64(currentHeap) / (1024 * 1024)
			bytesPerAdapter := float64(totalAllocated) / float64(len(adapters))
			sysMemMB := float64(m2.Sys) / (1024 * 1024)

			// Log detailed stats
			t.Logf("=== Memory Footprint Analysis ===")
			t.Logf("Adapters created: %d", len(adapters))
			t.Logf("Total allocated: %d bytes (%.2f MB)", totalAllocated, memUsedMB)
			t.Logf("Current heap usage: %d bytes (%.2f MB)", currentHeap, heapUsedMB)
			t.Logf("Bytes per adapter: %.2f", bytesPerAdapter)
			t.Logf("Heap objects: %d", m2.HeapObjects)
			t.Logf("System memory (from OS): %.2f MB", sysMemMB)
			t.Logf("Number of GC cycles: %d", m2.NumGC-m1.NumGC)

			// Project memory usage for larger scales based on heap usage
			if sc.projects == 500 {
				mem1000 := (heapUsedMB / 500) * 1000
				mem5000 := (heapUsedMB / 500) * 5000
				t.Logf("\n=== Projected Heap Memory Usage ===")
				t.Logf("1,000 projects: ~%.0f MB (~%.1f GB)", mem1000, mem1000/1024)
				t.Logf("5,000 projects: ~%.0f MB (~%.1f GB)", mem5000, mem5000/1024)
			}
		})
	}
}

// TestMemoryFootprint_WithListQueries measures memory usage when actually executing
// LIST queries against adapters, not just adding them. This simulates real-world
// usage where queries are executed and items are returned and cached.
//
// Memory Profiling:
//
//	To generate memory profiles for analysis:
//
//	1. Generate memory profile:
//	   go test -run TestMemoryFootprint_WithListQueries/35_projects \
//	     -memprofile=mem_35_projects.pprof ./discovery/...
//
//	2. Analyze the profile:
//	   go tool pprof mem_35_projects.pprof
//	   # Then use: top, list <function>, web, etc.
//
//	3. Or use web UI:
//	   go tool pprof -http=:8080 mem_35_projects.pprof
//	   # Then open http://localhost:8080 in browser
//
//	For heap profiles at specific points (after adapters, after queries):
//	   HEAP_PROFILE=heap go test -run TestMemoryFootprint_WithListQueries/35_projects -v ./discovery/...
func TestMemoryFootprint_WithListQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory footprint test with list queries in short mode")
	}

	scenarios := []struct {
		name     string
		projects int
		regions  int
		zones    int
		types    int
		timeout  time.Duration
	}{
		{"35_projects", 35, 35, 135, 88, 5 * time.Minute},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Force GC and get baseline
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			// Create adapters using BenchmarkListAdapter (returns 10 items per query)
			adapters := generateBenchmarkGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.types)
			engine, err := newBenchmarkEngine(adapters...)
			if err != nil {
				t.Fatalf("Failed to create engine: %v", err)
			}

			// Get memory stats after adding adapters (before queries)
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			// Write heap profile after adapters if requested
			if heapProfile := os.Getenv("HEAP_PROFILE"); heapProfile != "" {
				f, err := os.Create(fmt.Sprintf("%s_%s_%d_projects_after_adapters.pprof", heapProfile, sc.name, sc.projects))
				if err == nil {
					defer f.Close()
					runtime.GC()
					if err := pprof.WriteHeapProfile(f); err != nil {
						t.Logf("Failed to write heap profile: %v", err)
					} else {
						t.Logf("Heap profile (after adapters) written to: %s", f.Name())
					}
				}
			}

			// Execute LIST queries for each unique adapter type
			// This will expand to all matching scopes via ExpandQuery
			ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
			defer cancel()

			// Collect unique adapter types
			typeSet := make(map[string]bool)
			for _, adapter := range adapters {
				typeSet[adapter.Type()] = true
			}

			// Execute LIST queries for each unique adapter type across all scopes
			// This will expand to all matching scopes via ExpandQuery
			totalItems := 0
			totalErrors := 0

			// Execute one LIST query per adapter type (will expand to all scopes)
			for adapterType := range typeSet {
				query := &sdp.Query{
					Type:   adapterType,
					Scope:  "*", // Wildcard to match all scopes
					Method: sdp.QueryMethod_LIST,
				}

				items, _, errs, err := engine.executeQuerySync(ctx, query)
				if err != nil {
					t.Logf("Query execution error for type %s: %v", adapterType, err)
				}

				totalItems += len(items)
				totalErrors += len(errs)
			}

			// Get final memory stats after queries
			var m3 runtime.MemStats
			runtime.ReadMemStats(&m3)

			// Write heap profile if requested via environment variable
			if heapProfile := os.Getenv("HEAP_PROFILE"); heapProfile != "" {
				f, err := os.Create(fmt.Sprintf("%s_%s_%d_projects.pprof", heapProfile, sc.name, sc.projects))
				if err != nil {
					t.Logf("Failed to create heap profile: %v", err)
				} else {
					defer f.Close()
					runtime.GC() // Get accurate picture
					if err := pprof.WriteHeapProfile(f); err != nil {
						t.Logf("Failed to write heap profile: %v", err)
					} else {
						t.Logf("Heap profile written to: %s", f.Name())
					}
				}
			}

			// Calculate memory deltas
			allocAfterAdapters := m2.TotalAlloc - m1.TotalAlloc
			allocAfterQueries := m3.TotalAlloc - m2.TotalAlloc
			totalAllocated := m3.TotalAlloc - m1.TotalAlloc

			heapAfterAdapters := m2.HeapAlloc
			heapAfterQueries := m3.HeapAlloc

			// Convert to MB
			allocAfterAdaptersMB := float64(allocAfterAdapters) / (1024 * 1024)
			allocAfterQueriesMB := float64(allocAfterQueries) / (1024 * 1024)
			totalAllocatedMB := float64(totalAllocated) / (1024 * 1024)
			heapAfterAdaptersMB := float64(heapAfterAdapters) / (1024 * 1024)
			heapAfterQueriesMB := float64(heapAfterQueries) / (1024 * 1024)

			// Calculate per-item and per-adapter metrics
			bytesPerAdapter := float64(totalAllocated) / float64(len(adapters))
			bytesPerItem := float64(allocAfterQueries) / float64(totalItems)
			bytesPerProject := float64(totalAllocated) / float64(sc.projects)

			// Log detailed stats
			t.Logf("=== Memory Footprint Analysis with List Queries ===")
			t.Logf("Adapters created: %d", len(adapters))
			t.Logf("Adapter types queried: %d", len(typeSet))
			t.Logf("Total items returned: %d", totalItems)
			t.Logf("Total errors: %d", totalErrors)
			t.Logf("\n=== Memory After Adding Adapters ===")
			t.Logf("Total allocated: %d bytes (%.2f MB)", allocAfterAdapters, allocAfterAdaptersMB)
			t.Logf("Heap usage: %d bytes (%.2f MB)", heapAfterAdapters, heapAfterAdaptersMB)
			t.Logf("\n=== Memory After Executing Queries ===")
			t.Logf("Additional allocated: %d bytes (%.2f MB)", allocAfterQueries, allocAfterQueriesMB)
			t.Logf("Heap usage: %d bytes (%.2f MB)", heapAfterQueries, heapAfterQueriesMB)
			t.Logf("\n=== Total Memory Usage ===")
			t.Logf("Total allocated: %d bytes (%.2f MB)", totalAllocated, totalAllocatedMB)
			t.Logf("Bytes per adapter: %.2f", bytesPerAdapter)
			t.Logf("Bytes per item returned: %.2f", bytesPerItem)
			t.Logf("Bytes per project: %.2f", bytesPerProject)
			t.Logf("Heap objects: %d", m3.HeapObjects)
			t.Logf("System memory (from OS): %.2f MB", float64(m3.Sys)/(1024*1024))
			t.Logf("Number of GC cycles: %d", m3.NumGC-m1.NumGC)

			// Project memory usage for larger scales
			if sc.projects >= 100 {
				mem1000 := (heapAfterQueriesMB / float64(sc.projects)) * 1000
				mem5000 := (heapAfterQueriesMB / float64(sc.projects)) * 5000
				t.Logf("\n=== Projected Heap Memory Usage (with queries) ===")
				t.Logf("1,000 projects: ~%.0f MB (~%.1f GB)", mem1000, mem1000/1024)
				t.Logf("5,000 projects: ~%.0f MB (~%.1f GB)", mem5000, mem5000/1024)
			}
		})
	}
}

// BenchmarkMemoryFootprint_WithStats measures memory with runtime.MemStats
func BenchmarkMemoryFootprint_WithStats(b *testing.B) {
	scenarios := []struct {
		name     string
		projects int
		regions  int
		zones    int
		types    int
	}{
		{"Small_23proj", 23, 35, 135, 88},
		{"Medium_100proj", 100, 35, 135, 88},
		{"Large_500proj", 500, 35, 135, 88},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			for range b.N {
				b.StopTimer()

				// Get baseline memory
				runtime.GC()
				var m1 runtime.MemStats
				runtime.ReadMemStats(&m1)

				b.StartTimer()

				// Create and add adapters
				adapters := generateGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.types)
				sh := NewAdapterHost()
				err := sh.AddAdapters(adapters...)

				b.StopTimer()

				if err != nil {
					b.Fatal(err)
				}

				// Measure final memory (no GC to see actual usage)
				var m2 runtime.MemStats
				runtime.ReadMemStats(&m2)

				totalAllocated := m2.TotalAlloc - m1.TotalAlloc
				heapUsed := m2.HeapAlloc
				memUsedMB := float64(totalAllocated) / (1024 * 1024)
				heapUsedMB := float64(heapUsed) / (1024 * 1024)

				b.ReportMetric(float64(len(adapters)), "adapters")
				b.ReportMetric(memUsedMB, "total_alloc_MB")
				b.ReportMetric(heapUsedMB, "heap_MB")
				b.ReportMetric(float64(totalAllocated)/float64(len(adapters)), "bytes/adapter")
				b.ReportMetric(float64(m2.HeapObjects), "heap_objects")
				b.ReportMetric(float64(m2.Sys)/(1024*1024), "sys_memory_MB")
			}
		})
	}
}

// BenchmarkMemoryFootprint_WithListQueries measures memory usage when executing
// LIST queries against adapters that return 10 items each. This provides realistic
// memory consumption data for capacity planning when queries are actually executed.
func BenchmarkMemoryFootprint_WithListQueries(b *testing.B) {
	scenarios := []struct {
		name     string
		projects int
		regions  int
		zones    int
		types    int
	}{
		{"Medium_35proj", 35, 35, 135, 88},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			for range b.N {
				b.StopTimer()

				// Get baseline memory
				runtime.GC()
				var m1 runtime.MemStats
				runtime.ReadMemStats(&m1)

				// Create adapters using BenchmarkListAdapter (returns 10 items per query)
				adapters := generateBenchmarkGCPLikeAdapters(sc.projects, sc.regions, sc.zones, sc.types)
				engine, err := newBenchmarkEngine(adapters...)
				if err != nil {
					b.Fatalf("Failed to create engine: %v", err)
				}

				// Get memory after adding adapters
				var m2 runtime.MemStats
				runtime.ReadMemStats(&m2)

				b.StartTimer()

				// Execute LIST queries for each unique adapter type
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				// Collect unique adapter types
				typeSet := make(map[string]bool)
				for _, adapter := range adapters {
					typeSet[adapter.Type()] = true
				}

				totalItems := 0
				for adapterType := range typeSet {
					query := &sdp.Query{
						Type:   adapterType,
						Scope:  "*", // Wildcard to match all scopes
						Method: sdp.QueryMethod_LIST,
					}

					items, _, _, err := engine.executeQuerySync(ctx, query)
					if err != nil {
						// Log but don't fail - some queries might timeout in benchmarks
						b.Logf("Query execution error for type %s: %v", adapterType, err)
					}
					totalItems += len(items)
				}

				b.StopTimer()

				// Measure final memory after queries
				var m3 runtime.MemStats
				runtime.ReadMemStats(&m3)

				allocAfterAdapters := m2.TotalAlloc - m1.TotalAlloc
				allocAfterQueries := m3.TotalAlloc - m2.TotalAlloc
				totalAllocated := m3.TotalAlloc - m1.TotalAlloc
				heapAfterQueries := m3.HeapAlloc

				allocAfterAdaptersMB := float64(allocAfterAdapters) / (1024 * 1024)
				allocAfterQueriesMB := float64(allocAfterQueries) / (1024 * 1024)
				totalAllocatedMB := float64(totalAllocated) / (1024 * 1024)
				heapAfterQueriesMB := float64(heapAfterQueries) / (1024 * 1024)

				b.ReportMetric(float64(len(adapters)), "adapters")
				b.ReportMetric(float64(totalItems), "items_returned")
				b.ReportMetric(allocAfterAdaptersMB, "alloc_after_adapters_MB")
				b.ReportMetric(allocAfterQueriesMB, "alloc_after_queries_MB")
				b.ReportMetric(totalAllocatedMB, "total_alloc_MB")
				b.ReportMetric(heapAfterQueriesMB, "heap_MB")
				b.ReportMetric(float64(totalAllocated)/float64(len(adapters)), "bytes/adapter")
				b.ReportMetric(float64(allocAfterQueries)/float64(totalItems), "bytes/item")
				b.ReportMetric(float64(m3.HeapObjects), "heap_objects")
				b.ReportMetric(float64(m3.Sys)/(1024*1024), "sys_memory_MB")
			}
		})
	}
}
