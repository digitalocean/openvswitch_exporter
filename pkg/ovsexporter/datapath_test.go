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
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsnl"
)

func Test_datapathCollector(t *testing.T) {
	tests := []struct {
		name  string
		fn    func() ([]ovsnl.Datapath, error)
		empty bool
		out   []string
	}{
		{
			name: "none",
			fn: func() ([]ovsnl.Datapath, error) {
				return nil, nil
			},
			empty: true,
		},
		{
			name: "one",
			fn: func() ([]ovsnl.Datapath, error) {
				return []ovsnl.Datapath{{
					Index: 1,
					Name:  "ovs-system",
					Stats: ovsnl.DatapathStats{
						Hit: 1,
					},
				}}, nil
			},
			out: []string{
				`openvswitch_datapath_stats_hits_total{datapath="ovs-system"} 1`,
			},
		},
		{
			name: "multiple",
			fn: func() ([]ovsnl.Datapath, error) {
				return []ovsnl.Datapath{
					{
						Index: 1,
						Name:  "ovs-system",
						Stats: ovsnl.DatapathStats{
							Hit:    1,
							Missed: 2,
							Lost:   3,
							Flows:  4,
						},
						MegaflowStats: ovsnl.DatapathMegaflowStats{
							MaskHits: 5,
							Masks:    6,
						},
					},
					{
						Index: 2,
						Name:  "ovs-test",
						Stats: ovsnl.DatapathStats{
							Hit: 99,
						},
					},
				}, nil
			},
			out: []string{
				// Only bother to check that the hits total metric is present for
				// both datapaths, to reduce clutter here.
				`openvswitch_datapath_megaflow_stats_mask_hits_total{datapath="ovs-system"} 5`,
				`openvswitch_datapath_megaflow_stats_masks{datapath="ovs-system"} 6`,
				`openvswitch_datapath_stats_flows{datapath="ovs-system"} 4`,
				`openvswitch_datapath_stats_hits_total{datapath="ovs-system"} 1`,
				`openvswitch_datapath_stats_hits_total{datapath="ovs-test"} 99`,
				`openvswitch_datapath_stats_lost_total{datapath="ovs-system"} 3`,
				`openvswitch_datapath_stats_misses_total{datapath="ovs-system"} 2`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := testCollector(t, newDatapathCollector(tt.fn))

			if l := len(out); tt.empty && l != 0 {
				t.Fatalf("output should be empty, but was %d bytes", l)
			}

			for _, o := range tt.out {
				if !bytes.Contains(out, []byte(o)) {
					t.Fatalf("metrics output does not contain:\n\t%s\n\nfull output:\n\n%s", o, string(out))
				}
			}
		})
	}
}
