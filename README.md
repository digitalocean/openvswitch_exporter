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
