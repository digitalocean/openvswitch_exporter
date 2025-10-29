//go:build !linux
// +build !linux

// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"testing"

	"github.com/digitalocean/openvswitch_exporter/internal/conntrack"
)

// TestData represents test data for aggregator testing.
type TestData struct {
	Zone  uint16
	Mark  uint32
	Count int
}

// ValidateSnapshot compares a snapshot with expected test data, asserting counts match for non-zero entries.
func ValidateSnapshot(t *testing.T, snapshot map[conntrack.ZoneMarkKey]int, expected []TestData) {
	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}

	// Count non-zero entries in snapshot and expected.
	actualCount := 0
	for _, c := range snapshot {
		if c > 0 {
			actualCount++
		}
	}
	expectedCount := 0
	for _, d := range expected {
		if d.Count > 0 {
			expectedCount++
		}
	}

	if actualCount != expectedCount {
		t.Errorf("expected %d non-zero entries, got %d", expectedCount, actualCount)
	}

	// Validate specific entries.
	for _, d := range expected {
		if d.Count <= 0 {
			continue
		}
		key := conntrack.ZoneMarkKey{Zone: d.Zone, Mark: d.Mark}
		count, ok := snapshot[key]
		if !ok {
			t.Errorf("expected entry for key %v not found", key)
			continue
		}
		if count != d.Count {
			t.Errorf("expected count %d for key %v, got %d", d.Count, key, count)
		}
	}
}
