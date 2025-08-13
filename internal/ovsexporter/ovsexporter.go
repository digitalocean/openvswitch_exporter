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
	mu sync.Mutex
	cs []prometheus.Collector
}

var _ prometheus.Collector = &collector{}

// New creates a new Prometheus collector which collects metrics using the
// input Open vSwitch generic netlink client.
func New(c *ovsnl.Client) prometheus.Collector {
	collectors := []prometheus.Collector{
		newDatapathCollector(c.Datapath.List),
	}

	// Try to add conntrack collector, but don't fail if it's not available
	conntrackCollector := newConntrackCollector(func() ([]ovsnl.ConntrackEntry, error) {
		svc, err := ovsnl.NewConntrackService()
		if err != nil {
			return nil, err
		}
		defer svc.Close()
		return svc.List(context.Background())
	})

	// Test if conntrack service can be created
	if _, err := ovsnl.NewConntrackService(); err != nil {
		log.Printf("Warning: Conntrack service not available: %v. Conntrack metrics will be disabled.", err)
	} else {
		collectors = append(collectors, conntrackCollector)
		log.Printf("Conntrack collector enabled")
	}

	return &collector{
		cs: collectors,
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
