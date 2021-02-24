// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"fmt"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &datapathCollector{}

// A datapathCollector is a prometheus.Collector for Open vSwitch datapaths.
type datapathCollector struct {
	StatsHitsTotal             *prometheus.Desc
	StatsMissesTotal           *prometheus.Desc
	StatsLostTotal             *prometheus.Desc
	StatsFlows                 *prometheus.Desc
	MegaflowStatsMaskHitsTotal *prometheus.Desc
	MegaflowStatsMasks         *prometheus.Desc

	listDatapaths func() ([]ovsnl.Datapath, error)
}

// newDatapathCollector creates a prometheus.Collector for Open vSwitch
// datapaths.  The input function can be swapped for testing.
func newDatapathCollector(fn func() ([]ovsnl.Datapath, error)) prometheus.Collector {
	const (
		subsystem = "datapath"
	)

	var (
		labels = []string{"datapath"}
	)

	return &datapathCollector{
		StatsHitsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stats_hits_total"),
			"Number of flow table matches.",
			labels, nil,
		),

		StatsMissesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stats_misses_total"),
			"Number of flow table misses.",
			labels, nil,
		),

		StatsLostTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stats_lost_total"),
			"Number of flow table misses not sent to userspace.",
			labels, nil,
		),

		StatsFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stats_flows"),
			"Number of flows present.",
			labels, nil,
		),

		MegaflowStatsMaskHitsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "megaflow_stats_mask_hits_total"),
			"Number of megaflow masks used for flow lookups.",
			labels, nil,
		),

		MegaflowStatsMasks: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "megaflow_stats_masks"),
			"Number of megaflow masks present.",
			labels, nil,
		),

		listDatapaths: fn,
	}
}

// Describe implements prometheus.Collector.
func (c *datapathCollector) Describe(ch chan<- *prometheus.Desc) {
	dps := []*prometheus.Desc{
		c.StatsHitsTotal,
		c.StatsMissesTotal,
		c.StatsLostTotal,
		c.StatsFlows,
		c.MegaflowStatsMaskHitsTotal,
		c.MegaflowStatsMasks,
	}

	for _, dp := range dps {
		ch <- dp
	}
}

// Collect implements prometheus.Collector.
func (c *datapathCollector) Collect(ch chan<- prometheus.Metric) {
	dps, err := c.listDatapaths()
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.StatsHitsTotal, fmt.Errorf("error listing datapaths: %v", err))
		return
	}

	for _, dp := range dps {
		// Expose per-datapath metrics using statistics structures.
		tuples := []struct {
			desc      *prometheus.Desc
			valueType prometheus.ValueType
			value     uint64
		}{
			{
				desc:      c.StatsHitsTotal,
				valueType: prometheus.CounterValue,
				value:     dp.Stats.Hit,
			},
			{
				desc:      c.StatsMissesTotal,
				valueType: prometheus.CounterValue,
				value:     dp.Stats.Missed,
			},
			{
				desc:      c.StatsLostTotal,
				valueType: prometheus.CounterValue,
				value:     dp.Stats.Lost,
			},
			{
				desc:      c.StatsFlows,
				valueType: prometheus.GaugeValue,
				value:     dp.Stats.Flows,
			},
			{
				desc:      c.MegaflowStatsMaskHitsTotal,
				valueType: prometheus.CounterValue,
				value:     dp.MegaflowStats.MaskHits,
			},
			{
				desc:      c.MegaflowStatsMasks,
				valueType: prometheus.GaugeValue,
				value:     uint64(dp.MegaflowStats.Masks),
			},
		}

		for _, tuple := range tuples {
			// Label each metric with the datapath's name.
			ch <- prometheus.MustNewConstMetric(tuple.desc, tuple.valueType, float64(tuple.value), dp.Name)
		}
	}
}
