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
