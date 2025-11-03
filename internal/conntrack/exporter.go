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
	"fmt"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "openvswitch"
)

// Collector is a Prometheus collector for conntrack entries by zone and mark.
type Collector struct {
	desc *prometheus.Desc
	agg  MarkZoneAggregator
}

// Compile-time assertion that *Collector implements prometheus.Collector
var _ prometheus.Collector = (*Collector)(nil)

// NewCollector creates a new Prometheus collector for conntrack metrics.
// It starts the aggregator and returns a collector that can be registered
// with Prometheus. The aggregator must be stopped separately via Stop().
func NewCollector() (prometheus.Collector, MarkZoneAggregator, error) {
	agg, err := NewZoneMarkAggregator()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create zone/mark aggregator: %w", err)
	}

	if err := agg.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start zone/mark aggregator: %w", err)
	}

	return &Collector{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "conntrack", "entries"),
			"Number of conntrack entries by zone and mark",
			[]string{"zone", "mark"},
			nil,
		),
		agg: agg,
	}, agg, nil
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if c.agg == nil {
		log.Printf("No aggregator available, emitting zero metric")
		ch <- prometheus.MustNewConstMetric(
			c.desc,
			prometheus.GaugeValue,
			0,
			"unknown", "unknown",
		)
		return
	}

	snapshot := c.agg.Snapshot()
	for key, count := range snapshot {
		ch <- prometheus.MustNewConstMetric(
			c.desc,
			prometheus.GaugeValue,
			float64(count),
			fmt.Sprintf("%d", key.Zone),
			fmt.Sprintf("%d", key.Mark),
		)
	}
}
