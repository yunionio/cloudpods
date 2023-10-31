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
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestHelperWhere(t *testing.T) {
	Convey("Alert query where", t, func() {
		q := NewAlertCondition("", "").Query()
		w1 := q.Where()
		w1.Equal("hostname", "server1").NotEqual("hypervisor", "kvm").
			OR().GT("key", "val").LT("key1", "val2")
		So(w1.ToTags(), ShouldResemble, []monitor.MetricQueryTag{
			{
				Operator: "=",
				Key:      "hostname",
				Value:    "server1",
			},
			{
				Condition: "AND",
				Operator:  "!=",
				Key:       "hypervisor",
				Value:     "kvm",
			},
			{
				Condition: "OR",
				Operator:  ">",
				Key:       "key",
				Value:     "val",
			},
			{
				Condition: "OR",
				Operator:  "<",
				Key:       "key1",
				Value:     "val2",
			},
		})
	})
}

func TestHelperSelects(t *testing.T) {
	Convey("Alert query selects", t, func() {
		sels := NewAlertCondition("", "").Query().Selects()
		sels.Select("name").COUNT().MATH("/", "100")
		sels.Select("io_util")

		So(sels.parts[0].MetricQuerySelect, ShouldResemble, monitor.MetricQuerySelect{
			{
				Type:   "field",
				Params: []string{"name"},
			},
			{
				Type: "count",
			},
			{
				Type:   "math",
				Params: []string{"/ 100"},
			},
		})

		So(sels.parts[1].MetricQuerySelect, ShouldResemble, monitor.MetricQuerySelect{
			{
				Type:   "field",
				Params: []string{"io_util"},
			},
		})
	})
}

func TestAlertConfig(t *testing.T) {
	Convey("Alert config test", t, func() {
		enabled := true
		// disabled := false
		conf, err := NewAlertConfig("alert1", "5s", true)
		So(err, ShouldBeNil)
		q := conf.Condition("telegraf", "cpu").Avg().LT(50).Query()
		sels := q.Selects()
		sels.Select("usage_active").MEAN()
		sels.Select("usage_irq").COUNT()
		q.Where().Equal("host_ip", "10.168.222.231")
		So(conf.ToAlertCreateInput(), ShouldResemble, monitor.AlertCreateInput{
			Name:      "alert1",
			Frequency: 5,
			Settings: monitor.AlertSetting{
				Conditions: []monitor.AlertCondition{
					{
						Type: "query",
						Query: monitor.AlertQuery{
							Model: monitor.MetricQuery{
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
									{
										{
											Type:   "field",
											Params: []string{"usage_irq"},
										},
										{
											Type: "count",
										},
									},
								},
								Tags: []monitor.MetricQueryTag{
									{
										Key:      "host_ip",
										Operator: "=",
										Value:    "10.168.222.231",
									},
								},
							},
						},
						Reducer: monitor.Condition{
							Type: "avg",
						},
						Evaluator: monitor.Condition{
							Type:   "lt",
							Params: []float64{50},
						},
					},
				},
			},
			Enabled: &enabled,
			Level:   "",
		})
	})
}
