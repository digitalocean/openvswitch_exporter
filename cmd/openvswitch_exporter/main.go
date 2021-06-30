// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

// Command openvswitch_exporter implements a Prometheus exporter for Open vSwitch.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/digitalocean/openvswitch_exporter/internal/ovsexporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		metricsAddr = flag.String("metrics.addr", ":9310", "address for Open vSwitch exporter")
		metricsPath = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
	)

	flag.Parse()

	c, err := ovsnl.New()
	if err != nil {
		log.Fatalf("failed to connect to Open vSwitch datapath: %v", err)
	}
	defer c.Close()

	collector := ovsexporter.New(c)
	prometheus.MustRegister(collector)

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Printf("starting Open vSwitch exporter on %q", *metricsAddr)

	if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
		log.Fatalf("cannot start Open vSwitch exporter: %v", err)
	}
}
