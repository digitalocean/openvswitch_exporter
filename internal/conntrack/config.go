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

package conntrack

import (
	"time"
)

// Config holds configuration for the conntrack aggregator.
//
// The conntrack aggregator uses a default configuration system. The exporter currently
// uses default values for all conntrack settings. Custom configuration is supported
// programmatically via this Config struct and NewZoneMarkAggregatorWithConfig() function,
// but is not currently exposed at the top-level exporter interface.
//
// To use custom configuration, you would need to modify the exporter code to pass a
// custom Config struct instead of using NewZoneMarkAggregator() which always uses defaults.
type Config struct {
	// EventChanSize is the buffer size for the bounded event channel that receives
	// conntrack events from the netlink listener (default: 524288 = 512KB).
	// This channel acts as a buffer between the raw netlink events and the event
	// workers. When full, events are dropped and missedEvents counter is incremented.
	EventChanSize int

	// EventWorkerCount is the number of goroutines that process events from the
	// bounded event channel (default: 100). Each worker
	// processes NEW/DESTROY/UPDATE events.
	EventWorkerCount int

	// DestroyFlushIntvl is the base interval for flushing aggregated DESTROY deltas
	// into the main counts map (default: 50ms). The actual flushing is adaptive:
	// - >500K events/sec: 50ms intervals
	// - >100K events/sec: 100ms intervals
	// - >10K events/sec: 200ms intervals
	// - Normal: uses this configured interval
	// Faster flushing reduces latency but uses more CPU.
	DestroyFlushIntvl time.Duration

	// DestroyDeltaCap is the maximum number of DESTROY deltas that can be accumulated
	// before dropping events (default: 200000). DESTROY events are aggregated into
	// deltas to handle massive bursts without OOM. When this cap is reached, new
	// DESTROY events are dropped and missedEvents counter is incremented.
	DestroyDeltaCap int

	// DropsWarnThreshold is the threshold for triggering health check actions
	// (default: 10000). When missed events exceed this threshold, the health
	// monitor will attempt to restart the conntrack listener to recover from
	// potential connection issues.
	DropsWarnThreshold int64

	// ReadBufferSize is the socket read buffer size for the conntrack netlink
	// connection (default: 67108864 = 64MB). This affects how much data can be
	// buffered at the kernel level before being read by the application.
	ReadBufferSize int

	// WriteBufferSize is the socket write buffer size for the conntrack netlink
	// connection (default: 67108864 = 64MB). This affects how much data can be
	// buffered for writes to the kernel.
	WriteBufferSize int

	// HealthCheckIntvl is the interval for periodic health monitoring (default: 5m).
	// The health monitor checks missed events count and restarts the listener
	// if drops exceed DropsWarnThreshold.
	HealthCheckIntvl time.Duration

	// GracefulTimeout is the maximum time to wait for graceful shutdown of all
	// goroutines during Stop() (default: 30s). This includes waiting for event
	// workers to finish, flushing remaining deltas, and closing connections.
	GracefulTimeout time.Duration
}

// DefaultConfig returns default configuration values suitable for most production environments.
// This is used internally by NewZoneMarkAggregator().
//
// Default values:
//   - EventChanSize: 524288 (512KB)
//   - EventWorkerCount: 100
//   - DestroyFlushIntvl: 50ms
//   - DestroyDeltaCap: 200000
//   - DropsWarnThreshold: 10000
//   - ReadBufferSize: 67108864 (64MB)
//   - WriteBufferSize: 67108864 (64MB)
//   - HealthCheckIntvl: 5m
//   - GracefulTimeout: 30s
func DefaultConfig() *Config {
	return &Config{
		EventChanSize:      512 * 1024,
		EventWorkerCount:   100,
		DestroyFlushIntvl:  50 * time.Millisecond,
		DestroyDeltaCap:    200000,
		DropsWarnThreshold: 10000,
		ReadBufferSize:     64 * 1024 * 1024,
		WriteBufferSize:    64 * 1024 * 1024,
		HealthCheckIntvl:   5 * time.Minute,
		GracefulTimeout:    30 * time.Second,
	}
}
