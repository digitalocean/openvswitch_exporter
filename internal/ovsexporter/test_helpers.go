//go:build !linux
// +build !linux

// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"testing"
	"time"

	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
)

// TestHelper provides common testing utilities for conntrack tests
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// CreateMockAggregator creates a mock aggregator for testing
func (th *TestHelper) CreateMockAggregator() *conntrack.MockZoneMarkAggregator {
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		th.t.Fatalf("Failed to create mock aggregator: %v", err)
	}
	return agg
}

// CreateMockAggregatorWithData creates a mock aggregator with test data
func (th *TestHelper) CreateMockAggregatorWithData(data []TestData) *conntrack.MockZoneMarkAggregator {
	agg := th.CreateMockAggregator()

	for _, d := range data {
		agg.SetCount(d.Zone, d.Mark, d.Count)
	}

	return agg
}

// TestData represents test data for aggregator testing
type TestData struct {
	Zone  uint16
	Mark  uint32
	Count int
}

// CommonTestData provides commonly used test data sets
var CommonTestData = struct {
	Empty      []TestData
	Basic      []TestData
	Large      []TestData
	EdgeCases  []TestData
	Concurrent []TestData
}{
	Empty: []TestData{},

	Basic: []TestData{
		{Zone: 0, Mark: 100, Count: 1500},
		{Zone: 0, Mark: 200, Count: 2500},
		{Zone: 1, Mark: 300, Count: 3500},
	},

	Large: func() []TestData {
		var data []TestData
		for zone := uint16(0); zone < 10; zone++ {
			for mark := uint32(0); mark < 1000; mark++ {
				data = append(data, TestData{
					Zone:  zone,
					Mark:  mark,
					Count: int(uint32(zone)*1000 + mark),
				})
			}
		}
		return data
	}(),

	EdgeCases: []TestData{
		{Zone: 0, Mark: 0, Count: 0},                    // Zero values
		{Zone: 65535, Mark: 4294967295, Count: 1000000}, // Max values
		{Zone: 1, Mark: 1, Count: -1},                   // Negative count
	},

	Concurrent: []TestData{
		{Zone: 0, Mark: 100, Count: 100},
		{Zone: 1, Mark: 200, Count: 200},
		{Zone: 2, Mark: 300, Count: 300},
	},
}

// ValidateSnapshot validates a snapshot against expected data
func (th *TestHelper) ValidateSnapshot(snapshot map[conntrack.ZoneMarkKey]int, expected []TestData) {
	if snapshot == nil {
		th.t.Fatal("expected non-nil snapshot")
	}

	// Count non-zero entries
	actualCount := 0
	for _, count := range snapshot {
		if count > 0 {
			actualCount++
		}
	}

	expectedCount := 0
	for _, d := range expected {
		if d.Count > 0 {
			expectedCount++
		}
	}

	if actualCount != expectedCount {
		th.t.Errorf("expected %d non-zero entries, got %d", expectedCount, actualCount)
	}

	// Validate specific entries
	for _, d := range expected {
		if d.Count > 0 {
			key := conntrack.ZoneMarkKey{Zone: d.Zone, Mark: d.Mark}
			if count, exists := snapshot[key]; !exists {
				th.t.Errorf("expected entry for key %v not found", key)
			} else if count != d.Count {
				th.t.Errorf("expected count %d for key %v, got %d", d.Count, key, count)
			}
		}
	}
}

// RunConcurrentTest runs a test function concurrently
func (th *TestHelper) RunConcurrentTest(goroutines int, testFunc func()) {
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			testFunc()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// WaitForCondition waits for a condition to be true with timeout
func (th *TestHelper) WaitForCondition(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return true
			}
			if time.Now().After(deadline) {
				return false
			}
		}
	}
}

// BenchmarkCollector benchmarks collector performance
func (th *TestHelper) BenchmarkCollector(collector *conntrackCollector, iterations int) time.Duration {
	start := time.Now()

	for i := 0; i < iterations; i++ {
		snapshot := collector.agg.Snapshot()
		if snapshot == nil {
			th.t.Errorf("snapshot returned nil at iteration %d", i)
		}
	}

	return time.Since(start)
}

// TestCollectorWithData tests a collector with specific data
func (th *TestHelper) TestCollectorWithData(collector *conntrackCollector, expectedData []TestData) {
	// Test snapshot
	snapshot := collector.agg.Snapshot()
	th.ValidateSnapshot(snapshot, expectedData)
}

// TestCollectorLifecycle tests the full lifecycle of a collector
func (th *TestHelper) TestCollectorLifecycle(agg conntrack.Aggregator) {
	collector := newConntrackCollector(agg).(*conntrackCollector)

	// Test initial state
	if collector == nil {
		th.t.Fatal("expected non-nil collector")
	}

	// Test snapshot
	snapshot := collector.agg.Snapshot()
	if snapshot == nil {
		th.t.Fatal("expected non-nil snapshot")
	}

	// Test concurrent access
	th.RunConcurrentTest(10, func() {
		snapshot := collector.agg.Snapshot()
		if snapshot == nil {
			th.t.Error("concurrent snapshot returned nil")
		}
	})
}

// MockAggregatorBuilder provides a fluent interface for building mock aggregators
type MockAggregatorBuilder struct {
	agg *conntrack.MockZoneMarkAggregator
}

// NewMockAggregatorBuilder creates a new builder
func (th *TestHelper) NewMockAggregatorBuilder() *MockAggregatorBuilder {
	return &MockAggregatorBuilder{
		agg: th.CreateMockAggregator(),
	}
}

// WithData adds test data to the aggregator
func (b *MockAggregatorBuilder) WithData(data []TestData) *MockAggregatorBuilder {
	for _, d := range data {
		b.agg.SetCount(d.Zone, d.Mark, d.Count)
	}
	return b
}

// WithEntry adds a single entry to the aggregator
func (b *MockAggregatorBuilder) WithEntry(zone uint16, mark uint32, count int) *MockAggregatorBuilder {
	b.agg.SetCount(zone, mark, count)
	return b
}

// WithAddEntry adds an entry using AddEntry method
func (b *MockAggregatorBuilder) WithAddEntry(zone uint16, mark uint32) *MockAggregatorBuilder {
	b.agg.AddEntry(zone, mark)
	return b
}

// WithRemoveEntry removes an entry using RemoveEntry method
func (b *MockAggregatorBuilder) WithRemoveEntry(zone uint16, mark uint32) *MockAggregatorBuilder {
	b.agg.RemoveEntry(zone, mark)
	return b
}

// Build returns the built aggregator
func (b *MockAggregatorBuilder) Build() *conntrack.MockZoneMarkAggregator {
	return b.agg
}

// TestCollectorBuilder provides a fluent interface for building collectors
type TestCollectorBuilder struct {
	collector *conntrackCollector
}

// NewTestCollectorBuilder creates a new collector builder
func (th *TestHelper) NewTestCollectorBuilder() *TestCollectorBuilder {
	return &TestCollectorBuilder{}
}

// WithMockAggregator sets a mock aggregator
func (b *TestCollectorBuilder) WithMockAggregator(agg *conntrack.MockZoneMarkAggregator) *TestCollectorBuilder {
	b.collector = newConntrackCollector(agg).(*conntrackCollector)
	return b
}

// WithNilAggregator sets a nil aggregator
func (b *TestCollectorBuilder) WithNilAggregator() *TestCollectorBuilder {
	b.collector = newConntrackCollector(nil).(*conntrackCollector)
	return b
}

// Build returns the built collector
func (b *TestCollectorBuilder) Build() *conntrackCollector {
	return b.collector
}
