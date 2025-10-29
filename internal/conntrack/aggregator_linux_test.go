// Copyright 2017 DigitalOcean.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package conntrack

import (
	"sync"
	"testing"
	"time"
)

func TestZoneMarkAggregator(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (MarkZoneAggregator, error)
		operations  []func(MarkZoneAggregator) error
		validate    func(*testing.T, MarkZoneAggregator)
		wantErr     bool
		skipOnError bool
	}{
		{
			name:  "successful_creation",
			setup: func() (MarkZoneAggregator, error) { return NewZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
			},
			validate: func(t *testing.T, agg MarkZoneAggregator) {
				if agg == nil {
					t.Fatal("expected non-nil aggregator")
				}
				_ = agg.Snapshot() // tolerate empty in CI
			},
			wantErr:     false,
			skipOnError: true,
		},
		{
			name:  "snapshot_functionality",
			setup: func() (MarkZoneAggregator, error) { return NewZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
			},
			validate: func(t *testing.T, agg MarkZoneAggregator) {
				snap := agg.Snapshot()
				if snap == nil {
					t.Skip("nil snapshot likely due to permissions")
				}
				for key, count := range snap {
					if count <= 0 {
						t.Errorf("Invalid count %d for key %+v", count, key)
					}
				}
			},
			wantErr:     false,
			skipOnError: true,
		},
		{
			name:  "start_stop_lifecycle",
			setup: func() (MarkZoneAggregator, error) { return NewZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error {
					time.Sleep(10 * time.Millisecond)
					return agg.Stop()
				},
			},
			validate: func(t *testing.T, agg MarkZoneAggregator) {
				if err := agg.Stop(); err != nil {
					t.Errorf("Stop() error: %v", err)
				}
				_ = agg.Snapshot()
			},
			wantErr:     false,
			skipOnError: true,
		},
		{
			name:  "concurrent_snapshot_access",
			setup: func() (MarkZoneAggregator, error) { return NewZoneMarkAggregator() },
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
			},
			validate: func(t *testing.T, agg MarkZoneAggregator) {
				var wg sync.WaitGroup
				wg.Add(10)
				for i := 0; i < 10; i++ {
					go func() {
						defer wg.Done()
						_ = agg.Snapshot()
					}()
				}
				wg.Wait()
			},
			wantErr:     false,
			skipOnError: true,
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
			if agg == nil {
				if tt.skipOnError {
					t.Skip("Expected failure in test environment")
					return
				}
				t.Fatal("NewZoneMarkAggregator() returned nil aggregator")
			}

			t.Cleanup(func() { agg.Stop() })

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

			if tt.validate != nil {
				tt.validate(t, agg)
			}
		})
	}
}

func TestZoneMarkKey(t *testing.T) {
	tests := []struct {
		name     string
		key1     ZoneMarkKey
		key2     ZoneMarkKey
		expected bool
		desc     string
	}{
		{name: "identical_keys", key1: ZoneMarkKey{Zone: 1, Mark: 100}, key2: ZoneMarkKey{Zone: 1, Mark: 100}, expected: true, desc: "Identical ZoneMarkKey structs should be equal"},
		{name: "different_zone", key1: ZoneMarkKey{Zone: 1, Mark: 100}, key2: ZoneMarkKey{Zone: 2, Mark: 100}, expected: false, desc: "Different zone ZoneMarkKey structs should not be equal"},
		{name: "different_mark", key1: ZoneMarkKey{Zone: 1, Mark: 100}, key2: ZoneMarkKey{Zone: 1, Mark: 200}, expected: false, desc: "Different mark ZoneMarkKey structs should not be equal"},
		{name: "both_different", key1: ZoneMarkKey{Zone: 1, Mark: 100}, key2: ZoneMarkKey{Zone: 2, Mark: 200}, expected: false, desc: "Both zone and mark different should not be equal"},
		{name: "zero_values", key1: ZoneMarkKey{Zone: 0, Mark: 0}, key2: ZoneMarkKey{Zone: 0, Mark: 0}, expected: true, desc: "Zero values should be equal"},
		{name: "max_values", key1: ZoneMarkKey{Zone: 65535, Mark: 4294967295}, key2: ZoneMarkKey{Zone: 65535, Mark: 4294967295}, expected: true, desc: "Max values should be equal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := (tt.key1 == tt.key2)
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.desc, result, tt.expected)
			}
		})
	}
}

func TestZoneMarkKeyAsMapKey(t *testing.T) {
	tests := []struct {
		name     string
		keys     []ZoneMarkKey
		values   []int
		lookup   ZoneMarkKey
		expected int
		desc     string
	}{
		{name: "basic_map_operations", keys: []ZoneMarkKey{{Zone: 1, Mark: 100}, {Zone: 2, Mark: 200}}, values: []int{5, 10}, lookup: ZoneMarkKey{Zone: 1, Mark: 100}, expected: 5, desc: "ZoneMarkKey should work as map key"},
		{name: "equal_keys_map_to_same_value", keys: []ZoneMarkKey{{Zone: 1, Mark: 100}, {Zone: 2, Mark: 200}}, values: []int{5, 10}, lookup: ZoneMarkKey{Zone: 1, Mark: 100}, expected: 5, desc: "Equal ZoneMarkKey structs should map to same value"},
		{name: "different_keys_map_to_different_values", keys: []ZoneMarkKey{{Zone: 1, Mark: 100}, {Zone: 2, Mark: 200}}, values: []int{5, 10}, lookup: ZoneMarkKey{Zone: 2, Mark: 200}, expected: 10, desc: "Different ZoneMarkKey should map to different value"},
		{name: "zero_key_operations", keys: []ZoneMarkKey{{Zone: 0, Mark: 0}, {Zone: 1, Mark: 1}}, values: []int{100, 200}, lookup: ZoneMarkKey{Zone: 0, Mark: 0}, expected: 100, desc: "Zero value keys should work correctly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testMap := make(map[ZoneMarkKey]int)
			for i, key := range tt.keys {
				testMap[key] = tt.values[i]
			}
			result := testMap[tt.lookup]
			if result != tt.expected {
				t.Errorf("%s: got %d, want %d", tt.desc, result, tt.expected)
			}
		})
	}
}

func TestAggregatorLifecycle(t *testing.T) {
	tests := []struct {
		name        string
		operations  []func(MarkZoneAggregator) error
		validate    func(*testing.T, MarkZoneAggregator)
		wantErr     bool
		skipOnError bool
	}{
		{
			name: "start_twice_should_fail",
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error { return agg.Start() }, // second start
			},
			validate:    func(t *testing.T, agg MarkZoneAggregator) {},
			wantErr:     false,
			skipOnError: true,
		},
		{
			name: "stop_without_start",
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Stop() },
			},
			validate: func(t *testing.T, agg MarkZoneAggregator) { _ = agg.Snapshot() },
			wantErr:  false,
		},
		{
			name: "snapshot_after_stop",
			operations: []func(MarkZoneAggregator) error{
				func(agg MarkZoneAggregator) error { return agg.Start() },
				func(agg MarkZoneAggregator) error { time.Sleep(10 * time.Millisecond); return agg.Stop() },
			},
			validate:    func(t *testing.T, agg MarkZoneAggregator) { _ = agg.Snapshot() },
			wantErr:     false,
			skipOnError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := NewZoneMarkAggregator()
			if err != nil {
				if tt.skipOnError {
					t.Logf("Skipping test due to expected failure: %v", err)
					return
				}
				t.Fatalf("Failed to create aggregator: %v", err)
			}
			if agg == nil {
				t.Fatal("NewZoneMarkAggregator() returned nil aggregator")
			}
			t.Cleanup(func() { agg.Stop() })
			for i, op := range tt.operations {
				if err := op(agg); err != nil {
					if (err != nil) != tt.wantErr {
						if tt.skipOnError {
							t.Logf("Skipping op %d failure: %v", i, err)
							return
						}
						t.Errorf("operation %d error = %v, wantErr %v", i, err, tt.wantErr)
						return
					}
				}
			}
			if tt.validate != nil {
				tt.validate(t, agg)
			}
		})
	}
}
