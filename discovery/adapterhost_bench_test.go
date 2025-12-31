package discovery

import (
	"fmt"
	"runtime"
	"testing"
	"time"
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
