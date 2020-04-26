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
	ds := []*prometheus.Desc{
		c.StatsHitsTotal,
		c.StatsMissesTotal,
		c.StatsLostTotal,
		c.StatsFlows,
		c.MegaflowStatsMaskHitsTotal,
		c.MegaflowStatsMasks,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *datapathCollector) Collect(ch chan<- prometheus.Metric) {
	dps, err := c.listDatapaths()
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.StatsHitsTotal, fmt.Errorf("error listing datapaths: %v", err))
		return
	}

	for _, d := range dps {
		// Expose per-datapath metrics using statistics structures.
		tuples := []struct {
			d *prometheus.Desc
			t prometheus.ValueType
			v uint64
		}{
			{
				d: c.StatsHitsTotal,
				t: prometheus.CounterValue,
				v: d.Stats.Hit,
			},
			{
				d: c.StatsMissesTotal,
				t: prometheus.CounterValue,
				v: d.Stats.Missed,
			},
			{
				d: c.StatsLostTotal,
				t: prometheus.CounterValue,
				v: d.Stats.Lost,
			},
			{
				d: c.StatsFlows,
				t: prometheus.GaugeValue,
				v: d.Stats.Flows,
			},
			{
				d: c.MegaflowStatsMaskHitsTotal,
				t: prometheus.CounterValue,
				v: d.MegaflowStats.MaskHits,
			},
			{
				d: c.MegaflowStatsMasks,
				t: prometheus.GaugeValue,
				v: uint64(d.MegaflowStats.Masks),
			},
		}

		for _, t := range tuples {
			// Label each metric with the datapath's name.
			ch <- prometheus.MustNewConstMetric(t.d, t.t, float64(t.v), d.Name)
		}
	}
}
