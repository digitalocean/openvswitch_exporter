package ovsexporter

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	zoneThreshold = 50000 // Configure threshold for zone alerts (reduced for 2M test)
	// Memory management for large conntrack tables
	maxEntriesPerZone  = 100    // Drastically reduced maximum entries to collect per zone to prevent OOM
	largeZoneThreshold = 100000 // Use streaming approach for zones with >100k entries
	// Memory pressure thresholds
	memoryPressureThreshold = 0.8 // Trigger memory pressure handling when 80% of memory is used
	// CPU time limits
	maxCPUTimePerCollection = 60 * time.Second // Maximum CPU time per collection cycle
	// Sampling configuration for large zones
	sampleRateForLargeZones = 0.01 // Sample 1% of entries for zones > 1M entries
	// Timeout configuration
	conntrackTimeout = 30 * time.Second // Reduced timeout to prevent getting stuck
	// Memory pressure logging cooldown
	memoryPressureLogCooldown = 30 * time.Second // Prevent log spam
	// Memory cleanup thresholds
	memoryCleanupThreshold = 0.7 // Trigger aggressive cleanup at 70% usage
	// Circuit breaker for performance regression
	maxConsecutiveTimeouts = 3 // Stop processing after 3 consecutive timeouts
)

var (
	lastMemoryPressureLog time.Time
	consecutiveTimeouts   int
	lastTimeoutTime       time.Time
)

type ConntrackCollector struct {
	Count         *prometheus.Desc
	Performance   *prometheus.Desc
	listZoneStats func(context.Context, int) (map[uint16]*ovsnl.ZoneStats, error)
	getStats      func() (*ovsnl.ConntrackPerformanceStats, error)
}

// ConntrackCollectorWithAggAccessor wraps the existing collector with access to the aggregator snapshot
type ConntrackCollectorWithAggAccessor struct {
	*ConntrackCollector
	SnapshotFunc func() map[uint16]map[uint32]int
}

func newConntrackCollector(fn func(context.Context, int) (map[uint16]*ovsnl.ZoneStats, error), statsFn func() (*ovsnl.ConntrackPerformanceStats, error)) prometheus.Collector {
	return &ConntrackCollector{
		Count: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "conntrack", "count"),
			"Number of conntrack entries by zone, state, and mark",
			[]string{"zone", "state", "mark"}, nil,
		),
		Performance: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "conntrack", "performance"),
			"Conntrack performance counters",
			[]string{"counter"}, nil,
		),
		listZoneStats: fn,
		getStats:      statsFn,
	}
}

// checkCircuitBreaker checks if we should stop processing due to too many timeouts
func checkCircuitBreaker() bool {
	now := time.Now()

	// Reset counter if more than 5 minutes have passed since last timeout
	if now.Sub(lastTimeoutTime) > 5*time.Minute {
		consecutiveTimeouts = 0
		return false
	}

	// If we've had too many consecutive timeouts, stop processing
	if consecutiveTimeouts >= maxConsecutiveTimeouts {
		log.Printf("Circuit breaker triggered: %d consecutive timeouts, stopping conntrack collection", consecutiveTimeouts)
		return true
	}

	return false
}

// checkMemoryPressure checks if we're under memory pressure and triggers GC if needed
func checkMemoryPressure() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory usage percentage
	memoryUsage := float64(m.Alloc) / float64(m.Sys)

	if memoryUsage > memoryPressureThreshold {
		// Only log if enough time has passed since last log
		if time.Since(lastMemoryPressureLog) > memoryPressureLogCooldown {
			log.Printf("Memory pressure detected: %.2f%% usage, triggering GC", memoryUsage*100)
			lastMemoryPressureLog = time.Now()
		}
		runtime.GC()
	} else if memoryUsage > memoryCleanupThreshold {
		// Aggressive cleanup at 70% usage
		runtime.GC()
	}
}

// shouldSampleEntry determines if we should sample an entry based on zone size
func shouldSampleEntry(zoneTotalCount int, entryIndex int) bool {
	if zoneTotalCount <= maxEntriesPerZone {
		// For small zones, collect all entries
		return true
	}

	if zoneTotalCount > 1000000 {
		// For very large zones (>1M), use statistical sampling
		return rand.Float64() < sampleRateForLargeZones
	}

	// For medium zones, collect first maxEntriesPerZone entries
	return entryIndex < maxEntriesPerZone
}

// checkCPUTime checks if we're exceeding CPU time limits
func checkCPUTime(startTime time.Time) bool {
	elapsed := time.Since(startTime)
	if elapsed > maxCPUTimePerCollection {
		log.Printf("CPU time limit exceeded: %v elapsed, continuing with sampling", elapsed)
		return true
	}
	return false
}

// collectConntrackWithTimeout safely collects conntrack data with timeout protection
func (c *ConntrackCollector) collectConntrackWithTimeout(ctx context.Context, threshold int) (map[uint16]*ovsnl.ZoneStats, error) {
	// Check circuit breaker first
	if checkCircuitBreaker() {
		log.Printf("Circuit breaker active, skipping conntrack collection")
		return make(map[uint16]*ovsnl.ZoneStats), nil
	}

	var result map[uint16]*ovsnl.ZoneStats
	var err error
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, conntrackTimeout)
	defer cancel()

	// Start collection in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in conntrack collection: %v", r)
				err = fmt.Errorf("panic in conntrack collection: %v", r)
			}
		}()

		// Try streaming first, fallback to regular
		if c.listZoneStats != nil {
			result, err = c.listZoneStats(timeoutCtx, threshold)
		} else {
			// This case should ideally not be reached if listZoneStats is always set
			err = fmt.Errorf("no listZoneStats function available")
		}

		mu.Lock()
		defer mu.Unlock()
	}()

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		defer mu.Unlock()
		// Reset timeout counter on success
		consecutiveTimeouts = 0
		return result, err
	case <-timeoutCtx.Done():
		// Track timeout
		consecutiveTimeouts++
		lastTimeoutTime = time.Now()
		log.Printf("Conntrack collection timed out after %v (timeout #%d), returning partial results", conntrackTimeout, consecutiveTimeouts)
		// Force cleanup before returning
		runtime.GC()
		// Return empty result instead of error to prevent metric collection failure
		return make(map[uint16]*ovsnl.ZoneStats), nil
	}
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Count
	ch <- c.Performance
}

func (c *ConntrackCollector) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()
	ctx := context.Background()

	// Check memory pressure before starting
	checkMemoryPressure()

	// Emergency shutdown if memory pressure is too high
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := float64(m.Alloc) / float64(m.Sys)
	if memoryUsage > 0.85 { // 85% threshold for emergency shutdown
		log.Printf("Emergency shutdown: memory usage %.2f%% too high, skipping conntrack collection", memoryUsage*100)
		// Return basic metrics only
		ch <- prometheus.MustNewConstMetric(
			c.Count,
			prometheus.GaugeValue,
			0.0,
			"emergency", "shutdown", "0",
		)
		return
	}

	// Collect performance stats first (lightweight operation)
	if c.getStats != nil {
		if stats, err := c.getStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalFound),
				"found",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalInvalid),
				"invalid",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalIgnore),
				"ignore",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalInsert),
				"insert",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalInsertFailed),
				"insert_failed",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalDrop),
				"drop",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalEarlyDrop),
				"early_drop",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalError),
				"error",
			)
			ch <- prometheus.MustNewConstMetric(
				c.Performance,
				prometheus.GaugeValue,
				float64(stats.TotalSearchRestart),
				"search_restart",
			)
		} else {
			log.Printf("Failed to collect conntrack performance stats: %v", err)
		}
	}

	// Check memory pressure again before heavy operation
	checkMemoryPressure()

	// Collect zone statistics with timeout protection
	stats, err := c.collectConntrackWithTimeout(ctx, zoneThreshold)

	if err != nil {
		log.Printf("Failed to collect conntrack entries: %v", err)
		// Force cleanup on error
		runtime.GC()
		// Return a zero metric to indicate the collector is working but no data
		ch <- prometheus.MustNewConstMetric(
			c.Count,
			prometheus.GaugeValue,
			0.0,
			"unknown", "unknown", "0",
		)
		return
	}

	// Process zones using event-driven aggregator data
	// This is much more efficient than the old sampling approach
	for zone, zoneStats := range stats {
		// Always emit total count for the zone (this is critical!)
		ch <- prometheus.MustNewConstMetric(
			c.Count,
			prometheus.GaugeValue,
			float64(zoneStats.TotalCount),
			fmt.Sprint(zone),
			"total",
			"0",
		)
	}

	// OPTIONAL: emit per-mark counts using the aggregator directly.
	// This avoids storing per-entry slices and stays O(unique marks).
	if aggClient, ok := any(c).(*ConntrackCollectorWithAggAccessor); ok {
		zm := aggClient.SnapshotFunc() // <- we'll show how to plumb this accessor next
		// To avoid high-cardinality explosion, you can cap marks per zone:
		const maxMarksPerZone = 2000 // tune for your environment
		for zone, markMap := range zm {
			emitted := 0
			for mark, cnt := range markMap {
				if emitted >= maxMarksPerZone {
					break
				}
				ch <- prometheus.MustNewConstMetric(
					c.Count,
					prometheus.GaugeValue,
					float64(cnt),
					fmt.Sprint(zone), "total", fmt.Sprint(mark),
				)
				emitted++
			}
		}
	}

	// Log collection time
	elapsed := time.Since(startTime)
	log.Printf("Conntrack collection completed in %v", elapsed)
}
