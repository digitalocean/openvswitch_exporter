// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"testing"

	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
)

func TestConntrackCollector(t *testing.T) {
	// Create a mock aggregator
	agg, err := conntrack.NewZoneMarkAggregator()
	if err != nil {
		// This is expected to fail in test environment due to permission requirements
		t.Logf("Expected failure in test environment: NewZoneMarkAggregator() error = %v", err)
		// Test with nil aggregator to ensure collector handles gracefully
		collector := newConntrackCollector(nil)
		testCollector(t, collector)
		return
	}

	// Clean up aggregator after test
	t.Cleanup(agg.Stop)

	// Create collector with real aggregator
	collector := newConntrackCollector(agg)

	// Test the collector
	testCollector(t, collector)
}

func TestConntrackCollectorWithNilAggregator(t *testing.T) {
	// Test that the collector handles a nil aggregator gracefully
	collector := newConntrackCollector(nil)

	// This should not panic and should emit zero metrics
	testCollector(t, collector)
}
