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
	"os"
	"strconv"
	"time"
)

// ConntrackConfig holds configuration for the conntrack aggregator
type ConntrackConfig struct {
	EventChanSize      int
	EventWorkerCount   int
	DestroyFlushIntvl  time.Duration
	DestroyDeltaCap    int
	DropsWarnThreshold int64
	ReadBufferSize     int
	WriteBufferSize    int
	HealthCheckIntvl   time.Duration
	GracefulTimeout    time.Duration
}

// DefaultConntrackConfig returns default configuration values
func DefaultConntrackConfig() *ConntrackConfig {
	return &ConntrackConfig{
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

// LoadConntrackConfig loads conntrack configuration from environment variables
func LoadConntrackConfig() *ConntrackConfig {
	config := DefaultConntrackConfig()

	// Load from environment variables
	if size := os.Getenv("CONNTRACK_EVENT_CHAN_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			config.EventChanSize = s
		}
	}

	if count := os.Getenv("CONNTRACK_EVENT_WORKER_COUNT"); count != "" {
		if c, err := strconv.Atoi(count); err == nil && c > 0 {
			config.EventWorkerCount = c
		}
	}

	if interval := os.Getenv("CONNTRACK_DESTROY_FLUSH_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil && d > 0 {
			config.DestroyFlushIntvl = d
		}
	}

	if cap := os.Getenv("CONNTRACK_DESTROY_DELTA_CAP"); cap != "" {
		if c, err := strconv.Atoi(cap); err == nil && c > 0 {
			config.DestroyDeltaCap = c
		}
	}

	if threshold := os.Getenv("CONNTRACK_DROPS_WARN_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseInt(threshold, 10, 64); err == nil && t >= 0 {
			config.DropsWarnThreshold = t
		}
	}

	if size := os.Getenv("CONNTRACK_READ_BUFFER_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			config.ReadBufferSize = s
		}
	}

	if size := os.Getenv("CONNTRACK_WRITE_BUFFER_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			config.WriteBufferSize = s
		}
	}

	if interval := os.Getenv("CONNTRACK_HEALTH_CHECK_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil && d > 0 {
			config.HealthCheckIntvl = d
		}
	}

	if timeout := os.Getenv("CONNTRACK_GRACEFUL_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil && d > 0 {
			config.GracefulTimeout = d
		}
	}

	return config
}
