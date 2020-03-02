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

package influxdb

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func TestInfluxdbQueryParser(t *testing.T) {
	Convey("Influxdb query parser", t, func() {
		parser := &InfluxdbQueryParser{}
		Convey("can parse influxdb json model", func() {
			json := `
{
"group_by": [
  {
    "params": ["$interval"],
    "type": "time"
  },
  {
    "params": ["datacenter"],
    "type": "tag"
  },
  {
    "params": ["none"],
    "type": "fill"
  }
  ],
  "measurement": "logins.count",
  "tz": "Asia/Shanghai",
  "policy": "default",
  "refId": "B",
  "result_format": "time_series",
  "select": [
    [
      {
        "type": "field", 
        "params": ["value"]
      },
      {
        "type": "count", 
        "params": []
      }
    ],
    [
      {
        "type": "field", 
        "params": ["value"]
      },
      {
        "type": "bottom", 
        "params": ["3"]
      }
    ],
    [
      {
        "type": "field", 
        "params": ["value"]
      },
      {
        "type": "mean", 
        "params": []
      },
      {
        "type": "math", 
        "params": [" / 100"]
      }
    ]
  ],
  "alias": "serie alias",
  "tags": [
    {"key": "datacenter", "operator": "=", "value": "America"},
    {"condition": "OR", "key": "hostname", "operator": "=", "value": "server1"}
  ]
}
`
			obj, err := jsonutils.Parse([]byte(json))
			So(err, ShouldBeNil)
			apiQuery := new(tsdb.Query)
			So(obj.Unmarshal(apiQuery), ShouldBeNil)
			dsInfo := &tsdb.DataSource{TimeInterval: ">20s"}
			res, err := parser.Parse(apiQuery, dsInfo)
			So(err, ShouldBeNil)
			So(len(res.GroupBy), ShouldEqual, 3)
			So(len(res.Selects), ShouldEqual, 3)
			So(len(res.Tags), ShouldEqual, 2)
			So(res.Tz, ShouldEqual, "Asia/Shanghai")
			So(res.Interval, ShouldEqual, time.Second*20)
			So(res.Alias, ShouldEqual, "serie alias")
		})
	})
}
