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
	"yunion.io/x/jsonutils"

	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

type CommonAlertTerm struct {
	Database    string
	Measurement string
	Operator    string // and / or
	Field       []string
	FieldFunc   string

	Reduce        string
	Comparator    string
	Threshold     float64
	Filters       []monitorapi.MetricQueryTag
	FieldOpt      string
	Name          string
	ConditionType string
	From          string
	Interval      string
	GroupBy       string
	Level         string
}

func newCommonAlertQuery(tem *CommonAlertTerm) *monitorapi.CommonAlertQuery {
	mq := monitorapi.MetricQuery{
		Alias:        "",
		Tz:           "",
		Database:     tem.Database,
		Measurement:  tem.Measurement,
		Tags:         make([]monitorapi.MetricQueryTag, 0),
		GroupBy:      make([]monitorapi.MetricQueryPart, 0),
		Selects:      nil,
		Interval:     "",
		Policy:       "",
		ResultFormat: "",
	}

	for _, field := range tem.Field {
		sel := monitorapi.MetricQueryPart{
			Type:   "field",
			Params: []string{field},
		}
		selectPart := []monitorapi.MetricQueryPart{sel}
		if len(tem.FieldFunc) != 0 {
			selectPart = append(selectPart, monitorapi.MetricQueryPart{
				Type:   tem.FieldFunc,
				Params: []string{},
			})
		} else {
			selectPart = append(selectPart, monitorapi.MetricQueryPart{
				Type:   "mean",
				Params: []string{},
			})
		}
		mq.Selects = append(mq.Selects, selectPart)
	}
	if len(tem.Filters) != 0 {
		mq.Tags = append(mq.Tags, tem.Filters...)
	}

	alertQ := new(monitorapi.AlertQuery)
	alertQ.Model = mq
	alertQ.From = "60m"

	commonAlert := monitorapi.CommonAlertQuery{
		AlertQuery: alertQ,
		Reduce:     tem.Reduce,
		Comparator: tem.Comparator,
		Threshold:  tem.Threshold,
		Operator:   tem.Operator,
	}
	if tem.FieldOpt != "" {
		commonAlert.FieldOpt = monitorapi.CommonAlertFieldOpt_Division
	}
	if len(tem.ConditionType) != 0 {
		commonAlert.ConditionType = tem.ConditionType
	}
	if len(tem.GroupBy) != 0 {
		commonAlert.Model.GroupBy = append(commonAlert.Model.GroupBy, monitorapi.MetricQueryPart{
			Type:   "field",
			Params: []string{tem.GroupBy},
		})
	}
	return &commonAlert
}

var (
	cpuTem = &CommonAlertTerm{
		Operator:    "or",
		Database:    "telegraf",
		Measurement: "vm_cpu",
		Field:       []string{"usage_active"},
		Comparator:  ">=",
		Reduce:      "avg",
		Threshold:   50,
		Name:        "lzx-test.cpu.usage_active",
		Filters: []monitorapi.MetricQueryTag{
			{
				Key:      "id",
				Operator: "=",
				Value:    "a0eee5dd-3cfe-4ab1-8c79-aee1a8cf4dab",
			},
		},
	}
	memTem = &CommonAlertTerm{
		Operator:    "or",
		Database:    "telegraf",
		Measurement: "vm_mem",
		Field:       []string{"used_percent"},
		Comparator:  ">=",
		Reduce:      "avg",
		Threshold:   3,
		Name:        "lzx-test.mem.avaiable",
		Filters: []monitorapi.MetricQueryTag{
			{
				Key:      "id",
				Operator: "=",
				Value:    "a0eee5dd-3cfe-4ab1-8c79-aee1a8cf4dab",
			},
		},
	}
)

func init() {
	cmd := NewResourceCmd(modules.CommonAlerts)
	cmd.Create(new(options.CommonAlertCreateOptions))
	cmd.List(new(options.CommonAlertListOptions))
	cmd.Show(new(options.CommonAlertShowOptions))
	cmd.Perform("enable", &options.CommonAlertShowOptions{})
	cmd.Perform("disable", &options.CommonAlertShowOptions{})
	cmd.BatchDelete(new(options.CommonAlertDeleteOptions))
	cmd.Perform("config", &options.CommonAlertUpdateOptions{})

	type TestOpt struct {
		NAME     string
		RobotIds []string
		Users    []string
	}
	R(&TestOpt{}, "monitor-commonalert-create-mul-test", "create test monitor common alert", func(s *mcclient.ClientSession, opt *TestOpt) error {
		cpuQ := newCommonAlertQuery(cpuTem)
		memQ := newCommonAlertQuery(memTem)
		input := monitorapi.CommonAlertCreateInput{
			CommonMetricInputQuery: monitorapi.CommonMetricInputQuery{
				MetricQuery: []*monitorapi.CommonAlertQuery{
					cpuQ,
					memQ,
				},
			},
			AlertCreateInput: monitorapi.AlertCreateInput{
				Name: opt.NAME,
			},
			CommonAlertCreateBaseInput: monitorapi.CommonAlertCreateBaseInput{
				Recipients: opt.Users,
				RobotIds:   opt.RobotIds,
				Channel:    []string{"webconsole"},
				AlertType:  monitorapi.CommonAlertNomalAlertType,
				Scope:      "system",
			},
		}
		_, err := modules.CommonAlerts.Create(s, jsonutils.Marshal(input))
		return err
	})
}
