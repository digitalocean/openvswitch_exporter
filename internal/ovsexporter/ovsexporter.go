// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

// Package ovsexporter provides types used in the Open vSwitch Prometheus
// exporter.
package ovsexporter

import (
	"log"
	"sync"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
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
	aggregator       conntrack.Aggregator
}

// Make sure collector implements prometheus.Collector
var _ prometheus.Collector = &collector{}

// New creates a new Prometheus collector which collects metrics using the
// input Open vSwitch generic netlink client.
func New(c *ovsnl.Client) prometheus.Collector {
	collectors := []prometheus.Collector{
		newDatapathCollector(c.Datapath.List),
	}

	// Create the aggregator
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		log.Printf("Warning: Failed to create zone/mark aggregator: %v", err)
		return &collector{cs: collectors}
	}

	// Start the aggregator
	if err := agg.Start(); err != nil {
		log.Printf("Warning: Failed to start zone/mark aggregator: %v", err)
		return &collector{cs: collectors}
	}

	collectors = append(collectors, newConntrackCollector(agg))

	return &collector{
		cs:               collectors,
		conntrackEnabled: true,
		aggregator:       agg,
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

// Close cleans up resources with graceful shutdown
func (c *collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conntrackEnabled && c.aggregator != nil {
		if err := c.aggregator.Stop(); err != nil {
			log.Printf("Error stopping aggregator: %v", err)
			return err
		}
		log.Printf("Collector closed gracefully")
	}

	return nil
}
