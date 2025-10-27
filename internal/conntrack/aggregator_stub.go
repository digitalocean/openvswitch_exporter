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

//go:build !linux

package conntrack

import "fmt"

// NewZoneMarkAggregator returns an error on non-Linux platforms
func NewZoneMarkAggregator() (*ZoneMarkAggregator, error) {
	return nil, fmt.Errorf("conntrack aggregator is only supported on Linux")
}

// Start is a no-op on non-Linux platforms
func (a *ZoneMarkAggregator) Start() error {
	return fmt.Errorf("conntrack aggregator is only supported on Linux")
}

// Stop is a no-op on non-Linux platforms
func (a *ZoneMarkAggregator) Stop() {}

// Snapshot returns an empty map on non-Linux platforms
func (a *ZoneMarkAggregator) Snapshot() map[ZoneMarkKey]int {
	return make(map[ZoneMarkKey]int)
}

// RestartListener returns an error on non-Linux platforms
func (a *ZoneMarkAggregator) RestartListener() error {
	return fmt.Errorf("conntrack aggregator is only supported on Linux")
}
