// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

// Package ovsexporter provides types used in the Open vSwitch Prometheus
// exporter.
package ovsexporter

import (
	"context"
	"log"
	"sync"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "openvswitch"
)

// A collector aggregates Open vSwitch Prometheus collectors.
type collector struct {
	mu               sync.Mutex
	cs               []prometheus.Collector
	conntrackEnabled bool
}

var _ prometheus.Collector = &collector{}

// New creates a new Prometheus collector which collects metrics using the
// input Open vSwitch generic netlink client.
func New(c *ovsnl.Client) prometheus.Collector {
	collectors := []prometheus.Collector{
		newDatapathCollector(c.Datapath.List),
	}

	// When you build the collector in New(...):
	var snapshot func() map[uint16]map[uint32]int
	if c.Agg != nil {
		snapshot = c.Agg.Snapshot
	}
	base := newConntrackCollector(
		// listZoneStats:
		func(ctx context.Context, threshold int) (map[uint16]*ovsnl.ZoneStats, error) {
			if c.Agg == nil {
				return map[uint16]*ovsnl.ZoneStats{}, nil
			}
			zm := c.Agg.Snapshot()

			out := make(map[uint16]*ovsnl.ZoneStats, len(zm))
			for zone, marks := range zm {
				total := 0
				for _, cnt := range marks {
					total += cnt
				}
				// Always include the zone (so "total" time series is complete).
				zs := &ovsnl.ZoneStats{TotalCount: total}
				// No per-entry slice to avoid memory.
				// If you still want per-mark metrics, do it in Collect directly using zm.
				out[zone] = zs
				_ = threshold // threshold is not used here; you can still filter if desired.
			}
			return out, nil
		},
		// getStats: Disabled due to multicast connection issues
		nil, // This will skip stats collection entirely
	)
	conntrackCollector := &ConntrackCollectorWithAggAccessor{
		ConntrackCollector: base.(*ConntrackCollector),
		SnapshotFunc:       snapshot,
	}

	if c.Conntrack == nil {
		log.Printf("Warning: Conntrack service not available; metrics disabled.")
	} else {
		collectors = append(collectors, conntrackCollector)
		log.Printf("Conntrack collector enabled (event-driven)")
	}

	return &collector{cs: collectors, conntrackEnabled: true}
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
