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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MonitorResourceJointAlertOptions struct {
}

func (o *MonitorResourceJointAlertOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *MonitorResourceJointAlertOptions) Property() string {
	return "alert"
}

type MonitorResourceListOptions struct {
	options.BaseListOptions
	ResType     string    `help:"filter by resource type" json:"res_type"`
	ResId       []string  `help:"filter by resource id" json:"res_id"`
	ResName     string    `help:"filter by resource name" json:"res_name"`
	AlertStates []string  `help:"filter by alert state" json:"alert_states"`
	StartTime   time.Time `help:"start time for top query, format: 2025-01-01 00:00:00" json:"start_time"`
	EndTime     time.Time `help:"end time for top query, format: 2025-01-01 00:00:00" json:"end_time"`
	Top         int       `help:"return top N resources by alert count (default: 5)" json:"top"`
}

func (o *MonitorResourceListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type MonitorResourceDoActionOptions struct {
	ID     string `help:"ID of the resource"`
	ACTION string `help:"Action name"`
	Data   string `json:"json string data"`
}

func (o *MonitorResourceDoActionOptions) GetId() string {
	return o.ID
}

func (o *MonitorResourceDoActionOptions) Params() (jsonutils.JSONObject, error) {
	var data jsonutils.JSONObject = jsonutils.NewDict()
	if len(o.Data) != 0 {
		d, err := jsonutils.ParseString(o.Data)
		if err != nil {
			return nil, errors.Wrapf(err, "parse string to data: %s", o.Data)
		}
		data = d
	}
	input := monitor.MonitorResourceDoActionInput{
		Action: o.ACTION,
		Data:   data,
	}
	return jsonutils.Marshal(input), nil
}
