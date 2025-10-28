//go:build !linux
// +build !linux

package ovsexporter

import (
	"testing"

	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
)

func TestConntrackCollector(t *testing.T) {
	// Create a mock aggregator
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		t.Fatalf("Failed to create mock aggregator: %v", err)
	}

	// Clean up aggregator after test
	t.Cleanup(agg.Stop)

	// Add some test data
	agg.SetCount(0, 100, 1500)
	agg.SetCount(0, 200, 2500)
	agg.SetCount(1, 300, 3500)

	// Create collector with mock aggregator
	collector := newConntrackCollector(agg)

	// Test the collector
	testCollector(t, collector)
}

func TestConntrackCollectorWithNilAggregator(t *testing.T) {
	// Test that the collector handles a nil aggregator gracefully
	collector := newConntrackCollector(nil)

	// This should not panic and should emit zero metrics
	testCollector(t, collector)
}

func TestConntrackCollectorWithEmptyAggregator(t *testing.T) {
	// Create an empty mock aggregator
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		t.Fatalf("Failed to create mock aggregator: %v", err)
	}

	t.Cleanup(agg.Stop)

	// Create collector with empty aggregator
	collector := newConntrackCollector(agg)

	// Test the collector
	testCollector(t, collector)
}

func TestConntrackCollectorWithLargeDataset(t *testing.T) {
	// Create a mock aggregator with large dataset
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		t.Fatalf("Failed to create mock aggregator: %v", err)
	}

	t.Cleanup(agg.Stop)

	// Add large dataset
	// Simulate 2M entries across multiple zones
	for zone := uint16(0); zone < 10; zone++ {
		for mark := uint32(0); mark < 1000; mark++ {
			agg.SetCount(zone, mark, int(uint32(zone)*1000+mark))
		}
	}

	// Create collector
	collector := newConntrackCollector(agg)

	// Test the collector
	testCollector(t, collector)
}

func TestConntrackCollectorEdgeCases(t *testing.T) {
	// Test edge cases
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		t.Fatalf("Failed to create mock aggregator: %v", err)
	}

	t.Cleanup(agg.Stop)

	// Test zero values
	agg.SetCount(0, 0, 0)

	// Test maximum values
	agg.SetCount(65535, 4294967295, 1000000)

	// Test negative count (should be handled gracefully)
	agg.SetCount(1, 1, -1)

	collector := newConntrackCollector(agg)
	testCollector(t, collector)
}

func TestConntrackCollectorConcurrency(t *testing.T) {
	// Test concurrent access
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		t.Fatalf("Failed to create mock aggregator: %v", err)
	}

	t.Cleanup(agg.Stop)

	collector := newConntrackCollector(agg)

	// Test concurrent collection
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			testCollector(t, collector)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
