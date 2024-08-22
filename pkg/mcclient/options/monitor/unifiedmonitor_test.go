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

package monitor

import (
	"reflect"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestMetricQueryOptions_parseReducer(t *testing.T) {
	tests := []struct {
		reducer string
		want    *api.Condition
		wantErr bool
	}{
		{
			reducer: "avg",
			want: &api.Condition{
				Type: "avg",
			},
		},
		{
			reducer: "percentile(95)",
			want: &api.Condition{
				Type:   "percentile",
				Params: []float64{95},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.reducer, func(t *testing.T) {
			o := MetricQueryOptions{}
			got, err := o.parseReducer(tt.reducer)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReducer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseReducer() got = %v, want %v", got, tt.want)
			}
		})
	}
}
