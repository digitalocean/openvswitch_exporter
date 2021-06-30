// Copyright 2018-2021 DigitalOcean.
// SPDX-License-Identifier: Apache-2.0

package ovsexporter

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/util/promlint"
)

func testCollector(t *testing.T, collector prometheus.Collector) []byte {
	t.Helper()

	// Set up and gather metrics from a single pass.
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(collector); err != nil {
		t.Fatalf("failed to register Prometheus collector: %v", err)
	}

	srv := httptest.NewServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("failed to GET data from prometheus: %v", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read server response: %v", err)
	}

	// Check for lint cleanliness of metrics.
	problems, err := promlint.New(bytes.NewReader(buf)).Lint()
	if err != nil {
		t.Fatalf("failed to lint metrics: %v", err)
	}

	if len(problems) > 0 {
		for _, p := range problems {
			t.Logf("\t%s: %s", p.Metric, p.Text)
		}

		t.Fatal("failing test due to lint problems")
	}

	// Metrics check out, return to caller for further tests.
	return buf
}
