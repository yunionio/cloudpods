// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package victoriametrics

import (
	"testing"
)

func Test_newPointsByResults(t *testing.T) {
	tests := []struct {
		name    string
		results []ResponseDataResult
		want    []*points
		wantErr bool
	}{
		{
			name: "merge ok",
			results: []ResponseDataResult{
				{
					Metric: map[string]string{
						"__name__":         "disk_free",
						"__union_result__": "disk_free",
						"fstype":           "fuse.s3fs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "9223372036854775807"},
						[]interface{}{1699416000, "9223372036854775807"},
					},
				},
				{
					Metric: map[string]string{
						"__name__":         "disk_free",
						"__union_result__": "disk_free",
						"fstype":           "xfs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "277637316608"},
						[]interface{}{1699416000, "277599338496"},
					},
				},
				{
					Metric: map[string]string{
						"__name__":         "disk_total",
						"__union_result__": "disk_total",
						"fstype":           "fuse.s3fs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "9223372036854775807"},
						[]interface{}{1699416000, "9223372036854775807"},
					},
				},
				{
					Metric: map[string]string{
						"__name__":         "disk_total",
						"__union_result__": "disk_total",
						"fstype":           "xfs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "536608768000"},
						[]interface{}{1699416000, "536608768000"},
					},
				},
				{
					Metric: map[string]string{
						"__name__":         "disk_used",
						"__union_result__": "disk_used",
						"fstype":           "fuse.s3fs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "0"},
						[]interface{}{1699416000, "0"},
					},
				},
				{
					Metric: map[string]string{
						"__name__":         "disk_used",
						"__union_result__": "disk_used",
						"fstype":           "xfs",
					},
					Values: []ResponseDataResultValue{
						[]interface{}{1699414400, "258971451392"},
						[]interface{}{1699416000, "259009429504"},
					},
				},
			},
			want: []*points{
				{
					id:      "fstype->fuse.s3fs",
					columns: []string{"disk_free", "disk_total", "disk_used"},
					values: []ResponseDataResultValue{
						[]interface{}{1699414400, "9223372036854775807", "9223372036854775807", "0"},
						[]interface{}{1699416000, "9223372036854775807", "9223372036854775807", "0"},
					},
					tags: map[string]string{"fstype": "fuse.s3fs"},
				},
				{
					id:      "fstype->xfs",
					columns: []string{"disk_free", "disk_total", "disk_used"},
					values: []ResponseDataResultValue{
						[]interface{}{1699414400, "277637316608", "536608768000", "258971451392"},
						[]interface{}{1699416000, "277599338496", "536608768000", "259009429504"},
					},
					tags: map[string]string{"fstype": "xfs"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newPointsByResults(tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("newPointsByResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i := range got {
				gotP := got[i]
				wantP := tt.want[i]
				if gotP.isEqual(wantP) {
					t.Errorf("newPointsByResults() got = %v, want %v", gotP, wantP)
				}
			}
		})
	}
}
