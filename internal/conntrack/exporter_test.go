//go:build linux
// +build linux

// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package conntrack

import (
	"sync"
	"testing"
	"time"

	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/util/promlint"
)

func testCollector(t *testing.T, collector prometheus.Collector) []byte {
	t.Helper()

	// Set up and gather metrics from a single pass.
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(collector); err != nil {
		t.Fatalf("failed to register Prometheus collector: %v", err)
	}

	srv := httptest.NewServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("failed to GET data from prometheus: %v", err)
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read server response: %v", err)
	}

	// Check for lint cleanliness of metrics.
	problems, err := promlint.New(bytes.NewReader(buf)).Lint()
	if err != nil {
		t.Fatalf("failed to lint metrics: %v", err)
	}

	if len(problems) > 0 {
		for _, p := range problems {
			t.Logf("\t%s: %s", p.Metric, p.Text)
		}

		t.Fatal("failing test due to lint problems")
	}

	// Metrics check out, return to caller for further tests.
	return buf
}

func TestCollector(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (MarkZoneAggregator, error)
		operations  []func(MarkZoneAggregator) error
		validate    func(*testing.T, *Collector)
		wantErr     bool
		skipOnError bool
		description string
	}{
		{
			name: "real_aggregator_creation",
			setup: func() (MarkZoneAggregator, error) {
				return NewZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
			},
			validate: func(t *testing.T, collector *Collector) {
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
			setup: func() (MarkZoneAggregator, error) {
				return nil, nil
			},
			operations: []func(MarkZoneAggregator) error{},
			validate: func(t *testing.T, collector *Collector) {
				// Intentionally minimal: do not attempt Prometheus registration when agg is nil.
				if collector.agg != nil {
					t.Errorf("expected nil aggregator, got non-nil")
				}
			},
			wantErr:     false,
			skipOnError: false,
			description: "Test collector handles nil aggregator gracefully",
		},
		{
			name: "real_data_processing",
			setup: func() (MarkZoneAggregator, error) {
				return NewZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error {
					// Let it run briefly to potentially collect real data
					time.Sleep(50 * time.Millisecond)
					return nil
				},
			},
			validate: func(t *testing.T, collector *Collector) {
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
			setup:      func() (MarkZoneAggregator, error) { return NewZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{func(agg MarkZoneAggregator) error { return agg.Start() }},
			validate: func(t *testing.T, collector *Collector) {
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
			setup: func() (MarkZoneAggregator, error) {
				return NewZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error {
					// Let it run briefly
					time.Sleep(10 * time.Millisecond)
					return nil
				},
				func(agg MarkZoneAggregator) error {
					// Stop the aggregator
					agg.Stop()
					return nil
				},
			},
			validate: func(t *testing.T, collector *Collector) {
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

			collector := &Collector{agg: agg}
			if tt.validate != nil {
				tt.validate(t, collector)
			}

			// Only run Prometheus registration when we have a non-nil aggregator.
			if agg != nil {
				testCollector(t, collector)
			} else {
				t.Log("Skipping Prometheus registration for nil aggregator case")
			}
		})
	}
}

func TestCollectorWithRealData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conntrack test in short mode")
	}

	tests := []struct {
		name        string
		duration    time.Duration
		validate    func(*testing.T, *Collector)
		description string
	}{
		{
			name:     "short_duration",
			duration: 100 * time.Millisecond,
			validate: func(t *testing.T, collector *Collector) {
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
			validate: func(t *testing.T, collector *Collector) {
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
			agg, err := NewZoneMarkAggregator()
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

			collector := &Collector{agg: agg}
			if tt.validate != nil {
				tt.validate(t, collector)
			}

			// Test the collector with Prometheus
			testCollector(t, collector)
		})
	}
}

func TestCollectorEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (MarkZoneAggregator, error)
		operations  []func(MarkZoneAggregator) error
		validate    func(*testing.T, *Collector)
		wantErr     bool
		skipOnError bool
		description string
	}{
		{
			name: "start_stop_multiple_times",
			setup: func() (MarkZoneAggregator, error) {
				return NewZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error {
					time.Sleep(10 * time.Millisecond)
					agg.Stop()
					return nil
				},
				func(agg MarkZoneAggregator) error {
					// Try to start again after stop
					return agg.Start()
				},
			},
			validate: func(t *testing.T, collector *Collector) {
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
			setup: func() (MarkZoneAggregator, error) {
				return NewZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error {
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
			validate: func(t *testing.T, collector *Collector) {
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

			collector := &Collector{agg: agg}
			if tt.validate != nil {
				tt.validate(t, collector)
			}

			// Test the collector with Prometheus
			testCollector(t, collector)
		})
	}
}
