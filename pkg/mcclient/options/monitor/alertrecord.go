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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertRecordListOptions struct {
	options.BaseListOptions

	AlertId  string   `help:"id of alert"`
	Level    string   `help:"alert level"`
	State    string   `help:"alert state"`
	ResTypes []string `json:"res_types"`
	Alerting bool     `json:"alerting"`
}

func (o *AlertRecordListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertRecordShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertRecordShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertRecordShowOptions) GetId() string {
	return o.ID
}

type AlertRecordTotalOptions struct {
	ID string `help:"total-alert" json:"-"`
	options.BaseListOptions
}

func (o *AlertRecordTotalOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *AlertRecordTotalOptions) GetId() string {
	return o.ID
}
