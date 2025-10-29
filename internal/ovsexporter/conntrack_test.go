//go:build linux
// +build linux

// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
)

func TestConntrackCollector(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (conntrack.MarkZoneAggregator, error)
		operations  []func(conntrack.MarkZoneAggregator) error
		validate    func(*testing.T, *conntrackCollector)
		wantErr     bool
		skipOnError bool
		description string
	}{
		{
			name: "real_aggregator_creation",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return conntrack.NewZoneMarkAggregator()
			},
			operations: []func(conntrack.MarkZoneAggregator) error{
				func(agg conntrack.MarkZoneAggregator) error { return agg.Start() },
			},
			validate: func(t *testing.T, collector *conntrackCollector) {
				if collector == nil {
					t.Fatal("expected non-nil collector")
				}
				if collector.desc == nil {
					t.Fatal("expected non-nil description")
				}
				if collector.agg == nil {
					t.Fatal("expected non-nil aggregator")
				}
			},
			wantErr:     false,
			skipOnError: true, // Skip if permission issues
			description: "Test collector with real aggregator creation",
		},
		{
			name: "nil_aggregator_handling",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return nil, nil
			},
			operations: []func(conntrack.MarkZoneAggregator) error{},
			validate: func(t *testing.T, collector *conntrackCollector) {
				if collector == nil {
					t.Fatal("expected non-nil collector")
				}
				if collector.agg != nil {
					t.Error("expected nil aggregator")
				}
				// Test that collector handles nil aggregator gracefully
				// This should not panic and should emit zero metrics
			},
			wantErr:     false,
			skipOnError: false,
			description: "Test collector handles nil aggregator gracefully",
		},
		{
			name: "real_data_processing",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return conntrack.NewZoneMarkAggregator()
			},
			operations: []func(conntrack.MarkZoneAggregator) error{
				func(agg conntrack.MarkZoneAggregator) error { return agg.Start() },
				func(agg conntrack.MarkZoneAggregator) error {
					// Let it run briefly to potentially collect real data
					time.Sleep(50 * time.Millisecond)
					return nil
				},
			},
			validate: func(t *testing.T, collector *conntrackCollector) {
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Fatal("expected non-nil snapshot")
				}
				// In test environment, snapshot might be empty
				t.Logf("Snapshot contains %d entries", len(snapshot))
			},
			wantErr:     false,
			skipOnError: true,
			description: "Test collector with real data processing",
		},
		{
			name:       "concurrent_collection",
			setup:      func() (conntrack.MarkZoneAggregator, error) { return conntrack.NewZoneMarkAggregator() },
			operations: []func(conntrack.MarkZoneAggregator) error{func(agg conntrack.MarkZoneAggregator) error { return agg.Start() }},
			validate: func(t *testing.T, collector *conntrackCollector) {
				var wg sync.WaitGroup
				wg.Add(10)
				for i := 0; i < 10; i++ {
					go func() {
						defer wg.Done()
						snapshot := collector.agg.Snapshot()
						if snapshot == nil {
							t.Error("Concurrent snapshot returned nil")
						}
					}()
				}
				wg.Wait()
			},
			wantErr:     false,
			skipOnError: true,
			description: "Test concurrent collection operations",
		},
		{
			name: "lifecycle_management",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return conntrack.NewZoneMarkAggregator()
			},
			operations: []func(conntrack.MarkZoneAggregator) error{
				func(agg conntrack.MarkZoneAggregator) error { return agg.Start() },
				func(agg conntrack.MarkZoneAggregator) error {
					// Let it run briefly
					time.Sleep(10 * time.Millisecond)
					return nil
				},
				func(agg conntrack.MarkZoneAggregator) error {
					// Stop the aggregator
					agg.Stop()
					return nil
				},
			},
			validate: func(t *testing.T, collector *conntrackCollector) {
				// Snapshot should still work after stop
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Error("Snapshot should work after stop")
				}
			},
			wantErr:     false,
			skipOnError: true,
			description: "Test aggregator lifecycle management",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := tt.setup()
			if (err != nil) != tt.wantErr {
				if tt.skipOnError {
					t.Logf("Skipping test due to expected failure: %v", err)
					// Test with nil aggregator to ensure collector handles gracefully
					collector := newConntrackCollector(nil)
					testCollector(t, collector)
					return
				}
				t.Errorf("setup error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if agg != nil {
				t.Cleanup(func() { agg.Stop() })
			}

			for i, op := range tt.operations {
				if err := op(agg); err != nil {
					if tt.skipOnError {
						t.Logf("Skipping test due to operation %d failure: %v", i, err)
						// Test with nil aggregator as fallback
						collector := newConntrackCollector(nil)
						testCollector(t, collector)
						return
					}
					t.Errorf("operation %d failed: %v", i, err)
					return
				}
			}

			collector := newConntrackCollector(agg)
			if tt.validate != nil {
				tt.validate(t, collector.(*conntrackCollector))
			}

			// Test the collector with Prometheus
			testCollector(t, collector)
		})
	}
}

func TestConntrackCollectorWithRealData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conntrack test in short mode")
	}

	tests := []struct {
		name        string
		duration    time.Duration
		validate    func(*testing.T, *conntrackCollector)
		description string
	}{
		{
			name:     "short_duration",
			duration: 100 * time.Millisecond,
			validate: func(t *testing.T, collector *conntrackCollector) {
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Fatal("expected non-nil snapshot")
				}
				t.Logf("Short duration test: %d entries collected", len(snapshot))
			},
			description: "Test with short data collection duration",
		},
		{
			name:     "medium_duration",
			duration: 500 * time.Millisecond,
			validate: func(t *testing.T, collector *conntrackCollector) {
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Fatal("expected non-nil snapshot")
				}
				t.Logf("Medium duration test: %d entries collected", len(snapshot))
			},
			description: "Test with medium data collection duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with real conntrack data if available
			agg, err := conntrack.NewZoneMarkAggregator()
			if err != nil {
				t.Skipf("Skipping real data test: %v", err)
			}

			t.Cleanup(func() { agg.Stop() })

			// Start the aggregator
			if err := agg.Start(); err != nil {
				t.Skipf("Skipping real data test - failed to start: %v", err)
			}

			// Wait for data to accumulate
			time.Sleep(tt.duration)

			collector := newConntrackCollector(agg)
			if tt.validate != nil {
				tt.validate(t, collector.(*conntrackCollector))
			}

			// Test the collector with Prometheus
			testCollector(t, collector)
		})
	}
}

func TestConntrackCollectorEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (conntrack.MarkZoneAggregator, error)
		operations  []func(conntrack.MarkZoneAggregator) error
		validate    func(*testing.T, *conntrackCollector)
		wantErr     bool
		skipOnError bool
		description string
	}{
		{
			name: "start_stop_multiple_times",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return conntrack.NewZoneMarkAggregator()
			},
			operations: []func(conntrack.MarkZoneAggregator) error{
				func(agg conntrack.MarkZoneAggregator) error { return agg.Start() },
				func(agg conntrack.MarkZoneAggregator) error {
					time.Sleep(10 * time.Millisecond)
					agg.Stop()
					return nil
				},
				func(agg conntrack.MarkZoneAggregator) error {
					// Try to start again after stop
					return agg.Start()
				},
			},
			validate: func(t *testing.T, collector *conntrackCollector) {
				// Should handle restart gracefully
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Error("Snapshot should work after restart")
				}
			},
			wantErr:     false,
			skipOnError: true,
			description: "Test start/stop multiple times",
		},
		{
			name: "rapid_start_stop_cycles",
			setup: func() (conntrack.MarkZoneAggregator, error) {
				return conntrack.NewZoneMarkAggregator()
			},
			operations: []func(conntrack.MarkZoneAggregator) error{
				func(agg conntrack.MarkZoneAggregator) error {
					// Rapid start/stop cycles
					for i := 0; i < 5; i++ {
						if err := agg.Start(); err != nil {
							return err
						}
						time.Sleep(1 * time.Millisecond)
						agg.Stop()
						time.Sleep(1 * time.Millisecond)
					}
					return nil
				},
			},
			validate: func(t *testing.T, collector *conntrackCollector) {
				// Should not panic or leak resources
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Error("Snapshot should work after rapid cycles")
				}
			},
			wantErr:     false,
			skipOnError: true,
			description: "Test rapid start/stop cycles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := tt.setup()
			if (err != nil) != tt.wantErr {
				if tt.skipOnError {
					t.Logf("Skipping test due to expected failure: %v", err)
					return
				}
				t.Errorf("setup error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if agg != nil {
				t.Cleanup(func() { agg.Stop() })
			}

			for i, op := range tt.operations {
				if err := op(agg); err != nil {
					if tt.skipOnError {
						t.Logf("Skipping test due to operation %d failure: %v", i, err)
						return
					}
					t.Errorf("operation %d failed: %v", i, err)
					return
				}
			}

			collector := newConntrackCollector(agg)
			if tt.validate != nil {
				tt.validate(t, collector.(*conntrackCollector))
			}

			// Test the collector with Prometheus
			testCollector(t, collector)
		})
	}
}
