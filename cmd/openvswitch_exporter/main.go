// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

// Command openvswitch_exporter implements a Prometheus exporter for Open vSwitch.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
	"github.com/digitalocean/openvswitch_exporter/internal/ovsexporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		metricsAddr     = flag.String("metrics.addr", ":9310", "address for Open vSwitch exporter")
		metricsPath     = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
		enableConntrack = flag.Bool("enable.conntrack", true, "enable conntrack metrics exporter")
	)

	flag.Parse()

	c, err := ovsnl.New()
	if err != nil {
		log.Fatalf("failed to connect to Open vSwitch datapath: %v", err)
	}
	defer c.Close()

	collector := ovsexporter.New(c)
	prometheus.MustRegister(collector)

	// Optionally register conntrack collector
	if *enableConntrack {
		conntrackCollector, conntrackAggregator, err := conntrack.NewCollector()
		if err != nil {
			log.Printf("Warning: Failed to create conntrack collector: %v", err)
		} else {
			prometheus.MustRegister(conntrackCollector)
			defer func() {
				if conntrackAggregator != nil {
					if err := conntrackAggregator.Stop(); err != nil {
						log.Printf("Conntrack aggregator shutdown error: %v", err)
					}
				}
			}()
			log.Printf("Conntrack metrics exporter enabled")
		}
	}

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    *metricsAddr,
		Handler: mux,
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("starting Open vSwitch exporter on %q", *metricsAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("cannot start Open vSwitch exporter: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, stopping gracefully...", sig)

	// Graceful shutdown with 15 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("Exporter stopped")
}
