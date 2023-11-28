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

	"github.com/smartystreets/goconvey/convey"

	monitor2 "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
)

func TestAlertQuery(t *testing.T) {
	convey.Convey("Alert query test", t, func() {
		parser := new(InfluxdbQueryParser)
		q := monitor.NewAlertQuery("telegraf", "diskio").From("5m").To("now")
		q.Selects().Select("await").MEAN()
		q.Where().Equal("hostname", "host1").Equal("provider", "kvm")
		q.GroupBy().TAG("*").FILL_NULL()
		qCtx := q.ToTsdbQuery()
		influxdbQ, err := parser.Parse(qCtx.Queries[0], nil)
		convey.So(err, convey.ShouldBeNil)
		rawQuery, err := influxdbQ.Build(qCtx)
		convey.So(err, convey.ShouldBeNil)
		convey.So(rawQuery, convey.ShouldEqual, `SELECT mean("await") FROM "diskio" WHERE ("hostname" = 'host1' AND "provider" = 'kvm') AND time > now() - 5m GROUP BY * fill(null)`)
	})

	convey.Convey("Alert last query", t, func() {
		q := monitor.NewAlertQuery("telegraf", "diskio").From("5m").To("now")
		q.Selects().Select("*").LAST()
		q.Where().AddTag(&monitor2.MetricQueryTag{
			Operator: "=~",
			Key:      "project_id",
			Value:    "/xxx/",
		})
		tq := q.ToTsdbQuery()
		parser := new(InfluxdbQueryParser)
		influxQ, err := parser.Parse(tq.Queries[0], nil)
		convey.So(err, convey.ShouldBeNil)
		rawQ, err := influxQ.Build(tq)
		convey.So(err, convey.ShouldBeNil)
		convey.So(rawQ, convey.ShouldEqual, `SELECT last(*) FROM "diskio" WHERE "project_id" =~ /xxx/ AND time > now() - 5m`)
	})
}
