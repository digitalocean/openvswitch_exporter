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

// Config holds configuration for the conntrack aggregator
type Config struct {
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

// DefaultConfig returns default configuration values
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

// LoadConfig loads conntrack configuration from environment variables
