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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertPanelCreateOptions struct {
	apis.ScopedResourceCreateInput

	DashboardId string `help:"id of attached dashboard"`
	NAME        string `help:"Name of bashboard"`
	Metric      string `help:"Metric name, include measurement and field, e.g. vm_cpu.usage_active" required:"true"`
	Database    string
	Interval    string `help:"query aggregation interval e.g. 1m|5s"`
	From        string `help:"query start time e.g. 5m|6h"`
	To          string `help:"query end time"`
}

func (o *AlertPanelCreateOptions) Params() (jsonutils.JSONObject, error) {
	createInput := new(monitor.AlertPanelCreateInput)
	createInput.Name = o.NAME
	createInput.Scope = o.Scope
	createInput.From = o.From
	createInput.To = o.To
	createInput.DashboardId = o.DashboardId
	createInput.Interval = o.Interval
	alertQuery := new(monitor.CommonAlertQuery)
	metrics := strings.Split(o.Metric, ".")
	if len(metrics) != 2 {
		return nil, errors.Wrap(httperrors.ErrBadRequest, "metric")
	}
	measurement := metrics[0]
	field := metrics[1]
	sels := make([]monitor.MetricQuerySelect, 0)
	sels = append(sels, monitor.NewMetricQuerySelect(
		monitor.MetricQueryPart{
			Type:   "field",
			Params: []string{field},
		}))
	q := monitor.MetricQuery{
		Database:    o.Database,
		Measurement: measurement,
		Selects:     sels,
	}
	tmp := new(monitor.AlertQuery)
	tmp.Model = q
	alertQuery.AlertQuery = tmp
	createInput.MetricQuery = make([]*monitor.CommonAlertQuery, 0)
	createInput.MetricQuery = append(createInput.MetricQuery, alertQuery)
	return jsonutils.Marshal(createInput), nil
}

type AlertPanelListOptions struct {
	options.BaseListOptions
	DashboardId string `help:"id of attached dashboard"`
}

func (o *AlertPanelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertPanelShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertPanelShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertPanelShowOptions) GetId() string {
	return o.ID
}

type AlertPanelDeleteOptions struct {
	ID string `json:"-"`
}

func (o *AlertPanelDeleteOptions) GetId() string {
	return o.ID
}

func (o *AlertPanelDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
