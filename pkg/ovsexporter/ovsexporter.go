// Copyright 2018 DigitalOcean.
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

// Package ovsexporter provides types used in the Open vSwitch Prometheus
// exporter.
package ovsexporter

import (
	"sync"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "openvswitch"
)

// A collector aggregates Open vSwitch Prometheus collectors.
type collector struct {
	mu sync.Mutex
	cs []prometheus.Collector
}

var _ prometheus.Collector = &collector{}

// New creates a new Prometheus collector which collects metrics using the
// input Open vSwitch generic netlink client.
func New(c *ovsnl.Client) prometheus.Collector {
	return &collector{
		cs: []prometheus.Collector{
			// Additional generic netlink family collectors can be added here.
			newDatapathCollector(c.Datapath.List),
		},
	}
}

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cc := range c.cs {
		cc.Describe(ch)
	}
}

// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cc := range c.cs {
		cc.Collect(ch)
	}
}
