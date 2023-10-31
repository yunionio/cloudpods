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
