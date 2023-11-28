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

/*import (
	"context"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func Test_queryByRaw(t *testing.T) {
	ds := &tsdb.DataSource{
		Id:   "vm",
		Name: "vm",
		Type: "victoricmetrics",
		Url:  "http://192.168.222.171:34795/",
	}
	q := &tsdb.TsdbQuery{
		TimeRange: tsdb.NewTimeRange("48h", "now"),
		Queries: []*tsdb.Query{
			{
				RefId: "A",
				MetricQuery: monitor.MetricQuery{
					Database:    "telegraf",
					Measurement: "cpu",
					Selects: []monitor.MetricQuerySelect{
						{
							{
								Type:   "field",
								Params: []string{"usage_active"},
							},
							{
								Type: "mean",
							},
						},
					},
					Tags: []monitor.MetricQueryTag{
						{
							Key:      "res_type",
							Operator: "=",
							Value:    "host",
						},
					},
					GroupBy: []monitor.MetricQueryPart{
						//{
						//	Type:   "tag",
						//	Params: []string{"host_id"},
						//},
						{
							Type:   "tag",
							Params: []string{"*"},
						},
					},
				},
			},
		},
		Debug: false,
	}
	ep, _ := NewVMAdapter(ds)

	resp, err := ep.Query(context.Background(), ds, q)
	if err != nil {
		t.Fatalf("queryByRaw error: %v", err)
	}
	log.Infof("resp: %s", jsonutils.Marshal(resp).PrettyString())
}
*/
