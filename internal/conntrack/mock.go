//go:build !linux
// +build !linux

package conntrack

import (
	"context"
	"sync"
	"time"
)

// MockZoneMarkAggregator provides a mock implementation for non-Linux platforms
type MockZoneMarkAggregator struct {
	*ZoneMarkAggregator
	counts   map[ZoneMarkKey]int
	countsMu sync.RWMutex
}

// NewZoneMarkAggregator creates a mock aggregator for testing
func NewZoneMarkAggregator() (*MockZoneMarkAggregator, error) {
	return NewZoneMarkAggregatorWithConfig(LoadConfig())
}

// NewZoneMarkAggregatorWithConfig creates a mock aggregator with custom configuration
func NewZoneMarkAggregatorWithConfig(config *Config) (*MockZoneMarkAggregator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockZoneMarkAggregator{
		ZoneMarkAggregator: &ZoneMarkAggregator{
			config: config,
			ctx:    ctx,
			cancel: cancel,
		},
		counts: make(map[ZoneMarkKey]int),
	}, nil
}

// Snapshot returns a copy of the current counts
func (m *MockZoneMarkAggregator) Snapshot() map[ZoneMarkKey]int {
	m.countsMu.RLock()
	defer m.countsMu.RUnlock()

	snapshot := make(map[ZoneMarkKey]int)
	for k, v := range m.counts {
		snapshot[k] = v
	}
	return snapshot
}

// Start starts the mock aggregator (no-op for mock)
func (m *MockZoneMarkAggregator) Start() error {
	return nil
}

// Stop stops the mock aggregator with graceful shutdown
func (m *MockZoneMarkAggregator) Stop() error {
	m.cancel()
	// Mock implementation doesn't need actual cleanup
	return nil
}

// AddEntry adds a mock entry for testing
func (m *MockZoneMarkAggregator) AddEntry(zone uint16, mark uint32) {
	m.countsMu.Lock()
	defer m.countsMu.Unlock()

	key := ZoneMarkKey{Zone: zone, Mark: mark}
	m.counts[key]++
}

// RemoveEntry removes a mock entry for testing
func (m *MockZoneMarkAggregator) RemoveEntry(zone uint16, mark uint32) {
	m.countsMu.Lock()
	defer m.countsMu.Unlock()

	key := ZoneMarkKey{Zone: zone, Mark: mark}
	if m.counts[key] > 0 {
		m.counts[key]--
		if m.counts[key] == 0 {
			delete(m.counts, key)
		}
	}
}

// SetCount sets a specific count for testing
func (m *MockZoneMarkAggregator) SetCount(zone uint16, mark uint32, count int) {
	m.countsMu.Lock()
	defer m.countsMu.Unlock()

	key := ZoneMarkKey{Zone: zone, Mark: mark}
	if count <= 0 {
		delete(m.counts, key)
	} else {
		m.counts[key] = count
	}
}

// Clear clears all counts
func (m *MockZoneMarkAggregator) Clear() {
	m.countsMu.Lock()
	defer m.countsMu.Unlock()

	m.counts = make(map[ZoneMarkKey]int)
}

// GetEventRate returns a mock event rate
func (m *MockZoneMarkAggregator) GetEventRate() float64 {
	return 100.0 // Mock rate
}

// GetEventCount returns a mock event count
func (m *MockZoneMarkAggregator) GetEventCount() int64 {
	return int64(len(m.counts)) * 10 // Mock count
}

// GetMissedEvents returns a mock missed events count
func (m *MockZoneMarkAggregator) GetMissedEvents() int64 {
	return 0 // Mock no missed events
}

// IsHealthy returns true for mock
func (m *MockZoneMarkAggregator) IsHealthy() bool {
	return true
}

// GetLastEventTime returns current time for mock
func (m *MockZoneMarkAggregator) GetLastEventTime() time.Time {
	return time.Now()
}
