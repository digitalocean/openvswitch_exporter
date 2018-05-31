openvswitch_exporter [![Build Status](https://travis-ci.org/digitalocean/openvswitch_exporter.svg?branch=master)](https://travis-ci.org/digitalocean/openvswitch_exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/digitalocean/openvswitch_exporter)](https://goreportcard.com/report/github.com/digitalocean/openvswitch_exporter)
====================

Command `openvswitch_exporter` implements a Prometheus exporter for Open
vSwitch. Apache 2.0 licensed.

Usage
-----

Available flags for `openvswitch_exporter` include:

```none
$ ./openvswitch_exporter -h
Usage of ./openvswitch_exporter:
  -metrics.addr string
        address for Open vSwitch exporter (default ":9310")
  -metrics.path string
        URL path for surfacing collected metrics (default "/metrics")
```

Overview
--------

`openvswitch_exporter` currently exposes a variety of metrics related to the
Linux kernel Open vSwitch datapath, using the generic netlink `ovs_datapath`
family.

The exported metrics are similar to those found in `ovs-dpctl show`:

```none
hypervisor $ ovs-dpctl show | head -n 4
system@ovs-system:
        lookups: hit:111615762 missed:1312004 lost:0
        flows: 12
        masks: hit:151608278 total:8 hit/pkt:1.34
```

To see the metrics that are currently available, use `curl`:

```none
hypervisor $ curl -s http://localhost:9310/metrics | grep openvswitch
# HELP openvswitch_datapath_megaflow_stats_mask_hits_total Number of megaflow masks used for flow lookups.
# TYPE openvswitch_datapath_megaflow_stats_mask_hits_total counter
openvswitch_datapath_megaflow_stats_mask_hits_total{datapath="ovs-system"} 1.51606216e+08
# HELP openvswitch_datapath_megaflow_stats_masks Number of megaflow masks present.
# TYPE openvswitch_datapath_megaflow_stats_masks counter
openvswitch_datapath_megaflow_stats_masks{datapath="ovs-system"} 9
# HELP openvswitch_datapath_stats_flows Number of flows present.
# TYPE openvswitch_datapath_stats_flows gauge
openvswitch_datapath_stats_flows{datapath="ovs-system"} 21
# HELP openvswitch_datapath_stats_hits_total Number of flow table matches.
# TYPE openvswitch_datapath_stats_hits_total counter
openvswitch_datapath_stats_hits_total{datapath="ovs-system"} 1.11614549e+08
# HELP openvswitch_datapath_stats_lost_total Number of flow table misses not sent to userspace.
# TYPE openvswitch_datapath_stats_lost_total counter
openvswitch_datapath_stats_lost_total{datapath="ovs-system"} 0
# HELP openvswitch_datapath_stats_misses_total Number of flow table misses.
# TYPE openvswitch_datapath_stats_misses_total counter
openvswitch_datapath_stats_misses_total{datapath="ovs-system"} 1.311983e+06
```

To calculate the "hit/pkt" metric from `ovs-dpctl show` (average number of masks
visited per packet), you can use the following PromQL query:

```none
openvswitch_datapath_megaflow_stats_mask_hits_total / (openvswitch_datapath_stats_hits_total + openvswitch_datapath_stats_misses_total)
```
