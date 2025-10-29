//go:build !linux
// +build !linux

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
		description string
	}{
		{
			name: "basic_functionality",
			setup: func() (MarkZoneAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error {
					// Add test data
					mockAgg := agg.(*MockZoneMarkAggregator)
					mockAgg.SetCount(0, 100, 1500)
					mockAgg.SetCount(0, 200, 2500)
					mockAgg.SetCount(1, 300, 3500)
					return nil
				},
			},
			validate: func(t *testing.T, collector *Collector) {
				if collector == nil {
					t.Fatal("expected non-nil collector")
				}
				if collector.desc == nil {
					t.Fatal("expected non-nil description")
				}
			},
			wantErr:     false,
			description: "Test basic collector functionality with mock data",
		},
		{
			name: "nil_aggregator",
			setup: func() (MarkZoneAggregator, error) {
				return nil, nil
			},
			operations: []func(MarkZoneAggregator) error{},
			validate: func(t *testing.T, collector *Collector) {
				if collector == nil {
					t.Fatal("expected non-nil collector")
				}
				if collector.agg != nil {
					t.Error("expected nil aggregator")
				}
			},
			wantErr:     false,
			description: "Test collector handles nil aggregator gracefully",
		},
		{
			name: "empty_aggregator",
			setup: func() (MarkZoneAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{},
			validate: func(t *testing.T, collector *Collector) {
				if collector == nil {
					t.Fatal("expected non-nil collector")
				}
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Fatal("expected non-nil snapshot")
				}
				if len(snapshot) != 0 {
					t.Errorf("expected empty snapshot, got %d entries", len(snapshot))
				}
			},
			wantErr:     false,
			description: "Test collector with empty aggregator",
		},
		{
			name: "large_dataset",
			setup: func() (MarkZoneAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error {
					// Add large dataset - simulate 10K entries across multiple zones
					mockAgg := agg.(*MockZoneMarkAggregator)
					for zone := uint16(0); zone < 10; zone++ {
						for mark := uint32(0); mark < 1000; mark++ {
							mockAgg.SetCount(zone, mark, int(uint32(zone)*1000+mark))
						}
					}
					return nil
				},
			},
			validate: func(t *testing.T, collector *Collector) {
				snapshot := collector.agg.Snapshot()
				if len(snapshot) != 10000 {
					t.Errorf("expected 10000 entries, got %d", len(snapshot))
				}
			},
			wantErr:     false,
			description: "Test collector with large dataset",
		},
		{
			name: "edge_cases",
			setup: func() (MarkZoneAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error {
					mockAgg := agg.(*MockZoneMarkAggregator)
					// Test zero values
					mockAgg.SetCount(0, 0, 0)
					// Test maximum values
					mockAgg.SetCount(65535, 4294967295, 1000000)
					// Test negative count (should be handled gracefully)
					mockAgg.SetCount(1, 1, -1)
					return nil
				},
			},
			validate: func(t *testing.T, collector *Collector) {
				snapshot := collector.agg.Snapshot()
				// Should have 2 entries (zero and negative counts should be filtered out)
				if len(snapshot) != 2 {
					t.Errorf("expected 2 entries, got %d", len(snapshot))
				}
			},
			wantErr:     false,
			description: "Test collector with edge cases",
		},
		{
			name:  "concurrent_operations",
			setup: func() (MarkZoneAggregator, error) { return NewMockZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error {
					mockAgg := agg.(*MockZoneMarkAggregator)
					mockAgg.SetCount(0, 100, 100)
					return nil
				},
			},
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
			description: "Test concurrent collector operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := tt.setup()
			if (err != nil) != tt.wantErr {
				t.Errorf("setup error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if agg != nil {
				t.Cleanup(func() { agg.Stop() })
			}

			for i, op := range tt.operations {
				if err := op(agg); err != nil {
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

func TestMockAggregatorOperations(t *testing.T) {
	tests := []struct {
		name        string
		operations  []func(*MockZoneMarkAggregator)
		validate    func(*testing.T, *MockZoneMarkAggregator)
		description string
	}{
		{
			name: "add_remove_entries",
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					agg.AddEntry(0, 100)
					agg.AddEntry(0, 100)
					agg.AddEntry(1, 200)
				},
				func(agg *MockZoneMarkAggregator) {
					agg.RemoveEntry(0, 100)
				},
			},
			validate: func(t *testing.T, agg *MockZoneMarkAggregator) {
				snapshot := agg.Snapshot()
				if len(snapshot) != 2 {
					t.Errorf("expected 2 entries, got %d", len(snapshot))
				}
				// Check specific counts
				key1 := ZoneMarkKey{Zone: 0, Mark: 100}
				key2 := ZoneMarkKey{Zone: 1, Mark: 200}
				if snapshot[key1] != 1 {
					t.Errorf("expected count 1 for key %v, got %d", key1, snapshot[key1])
				}
				if snapshot[key2] != 1 {
					t.Errorf("expected count 1 for key %v, got %d", key2, snapshot[key2])
				}
			},
			description: "Test add/remove entry operations",
		},
		{
			name: "set_count_operations",
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					agg.SetCount(0, 100, 1500)
					agg.SetCount(1, 200, 2500)
					agg.SetCount(2, 300, 0) // Should be filtered out
				},
			},
			validate: func(t *testing.T, agg *MockZoneMarkAggregator) {
				snapshot := agg.Snapshot()
				if len(snapshot) != 2 {
					t.Errorf("expected 2 entries, got %d", len(snapshot))
				}
				key1 := ZoneMarkKey{Zone: 0, Mark: 100}
				key2 := ZoneMarkKey{Zone: 1, Mark: 200}
				if snapshot[key1] != 1500 {
					t.Errorf("expected count 1500 for key %v, got %d", key1, snapshot[key1])
				}
				if snapshot[key2] != 2500 {
					t.Errorf("expected count 2500 for key %v, got %d", key2, snapshot[key2])
				}
			},
			description: "Test set count operations",
		},
		{
			name: "clear_operations",
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					agg.SetCount(0, 100, 1500)
					agg.SetCount(1, 200, 2500)
				},
				func(agg *MockZoneMarkAggregator) {
					agg.Clear()
				},
			},
			validate: func(t *testing.T, agg *MockZoneMarkAggregator) {
				snapshot := agg.Snapshot()
				if len(snapshot) != 0 {
					t.Errorf("expected empty snapshot after clear, got %d entries", len(snapshot))
				}
			},
			description: "Test clear operations",
		},
		{
			name: "health_metrics",
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					agg.SetCount(0, 100, 1000)
				},
			},
			validate: func(t *testing.T, agg *MockZoneMarkAggregator) {
				if !agg.IsHealthy() {
					t.Error("expected healthy aggregator")
				}
				if agg.GetEventRate() != 100.0 {
					t.Errorf("expected event rate 100.0, got %f", agg.GetEventRate())
				}
				if agg.GetMissedEvents() != 0 {
					t.Errorf("expected 0 missed events, got %d", agg.GetMissedEvents())
				}
				lastEventTime := agg.GetLastEventTime()
				if lastEventTime.IsZero() {
					t.Error("expected non-zero last event time")
				}
			},
			description: "Test health metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := NewMockZoneMarkAggregator()
			if err != nil {
				t.Fatalf("Failed to create mock aggregator: %v", err)
			}
			t.Cleanup(func() { agg.Stop() })

			for _, op := range tt.operations {
				op(agg)
				// Add small delay between operations to test timing
				time.Sleep(1 * time.Millisecond)
			}

			if tt.validate != nil {
				tt.validate(t, agg)
			}
		})
	}
}

func TestCollectorIntegration(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (*MockZoneMarkAggregator, error)
		operations  []func(*MockZoneMarkAggregator)
		validate    func(*testing.T, *Collector)
		description string
	}{
		{
			name: "full_lifecycle",
			setup: func() (*MockZoneMarkAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					// Start the aggregator
					agg.Start()
				},
				func(agg *MockZoneMarkAggregator) {
					// Add some data
					agg.SetCount(0, 100, 1500)
					agg.SetCount(1, 200, 2500)
				},
				func(agg *MockZoneMarkAggregator) {
					// Modify data
					agg.AddEntry(0, 100)
					agg.RemoveEntry(1, 200)
				},
			},
			validate: func(t *testing.T, collector *Collector) {
				// Test that collector can handle the aggregator
				snapshot := collector.agg.Snapshot()
				if snapshot == nil {
					t.Fatal("expected non-nil snapshot")
				}
				// Should have 2 entries (one added, one removed)
				if len(snapshot) != 2 {
					t.Errorf("expected 2 entries, got %d", len(snapshot))
				}
			},
			description: "Test full lifecycle with collector",
		},
		{
			name: "stress_test",
			setup: func() (*MockZoneMarkAggregator, error) {
				return NewMockZoneMarkAggregator()
			},
			operations: []func(*MockZoneMarkAggregator){
				func(agg *MockZoneMarkAggregator) {
					// Add many entries rapidly
					for i := 0; i < 1000; i++ {
						agg.SetCount(uint16(i%10), uint32(i), i)
					}
				},
			},
			validate: func(t *testing.T, collector *Collector) {
				snapshot := collector.agg.Snapshot()
				if len(snapshot) != 1000 {
					t.Errorf("expected 1000 entries, got %d", len(snapshot))
				}
			},
			description: "Test stress scenario with many entries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := tt.setup()
			if err != nil {
				t.Fatalf("Failed to create mock aggregator: %v", err)
			}
			t.Cleanup(func() { agg.Stop() })

			for _, op := range tt.operations {
				op(agg)
				// Small delay between operations
				time.Sleep(1 * time.Millisecond)
			}

			collector := &Collector{agg: agg}
			if tt.validate != nil {
				tt.validate(t, collector)
			}

			// Test with Prometheus
			testCollector(t, collector)
		})
	}
}
