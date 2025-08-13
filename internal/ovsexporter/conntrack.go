package ovsexporter

import (
	"fmt"
	"log"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

type conntrackCollector struct {
	Count                *prometheus.Desc
	listConntrackEntries func() ([]ovsnl.ConntrackEntry, error)
}

func newConntrackCollector(fn func() ([]ovsnl.ConntrackEntry, error)) prometheus.Collector {
	return &conntrackCollector{
		Count: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "conntrack", "count"),
			"Number of conntrack entries by zone, state, and mark",
			[]string{"zone", "state", "mark"}, nil,
		),
		listConntrackEntries: fn,
	}
}

func (c *conntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Count
}

func (c *conntrackCollector) Collect(ch chan<- prometheus.Metric) {
	entries, err := c.listConntrackEntries()
	if err != nil {
		log.Printf("Failed to collect conntrack entries: %v", err)
		// Return a zero metric to indicate the collector is working but no data
		ch <- prometheus.MustNewConstMetric(
			c.Count,
			prometheus.GaugeValue,
			0.0,
			"unknown", "unknown", "0",
		)
		return
	}

	// Log the number of entries found for debugging
	log.Printf("Found %d conntrack entries", len(entries))

	// Aggregate counts
	counts := make(map[string]map[string]map[string]int)
	for _, e := range entries {
		zone := fmt.Sprintf("%d", e.Zone)
		state := e.State
		mark := fmt.Sprintf("%d", e.Mark)
		if counts[zone] == nil {
			counts[zone] = make(map[string]map[string]int)
		}
		if counts[zone][state] == nil {
			counts[zone][state] = make(map[string]int)
		}
		counts[zone][state][mark]++
	}

	for zone, stateMap := range counts {
		for state, markMap := range stateMap {
			for mark, count := range markMap {
				ch <- prometheus.MustNewConstMetric(
					c.Count,
					prometheus.GaugeValue,
					float64(count),
					zone, state, mark,
				)
			}
		}
	}
}
