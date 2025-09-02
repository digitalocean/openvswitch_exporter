package ovsexporter

import (
	"fmt"
	"log"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/prometheus/client_golang/prometheus"
)

type conntrackCollector struct {
	desc *prometheus.Desc
	agg  *ovsnl.ZoneMarkAggregator
}

// ConntrackCollectorWithAggAccessor wraps the existing collector with access to the aggregator snapshot
type ConntrackCollectorWithAggAccessor struct {
	*conntrackCollector
	SnapshotFunc func() map[uint16]map[uint32]int
}

func newConntrackCollector(agg *ovsnl.ZoneMarkAggregator) prometheus.Collector {
	return &conntrackCollector{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "conntrack", "count"),
			"Number of conntrack entries by zone and mark",
			[]string{"zone", "mark"},
			nil,
		),
		agg: agg,
	}
}

func (c *conntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *conntrackCollector) Collect(ch chan<- prometheus.Metric) {
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
	for zone, marks := range snapshot {
		for mark, count := range marks {
			ch <- prometheus.MustNewConstMetric(
				c.desc,
				prometheus.GaugeValue,
				float64(count),
				fmt.Sprintf("%d", zone),
				fmt.Sprintf("%d", mark),
			)
		}
	}
}
