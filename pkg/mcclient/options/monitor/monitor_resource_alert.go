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

type MonitorResourceAlertListOptions struct {
	options.BaseListOptions
	MonitorResourceId string `help:"ID of monitor resource" json:"monitor_resource_id"`
	AlertId           string `help:"ID  of alert" json:"alert_id"`
	Alerting          bool   `help:"search alerting resource" json:"alerting"`
	SendState         string `json:"send_state"`
	AllState          bool   `help:"Show all state" json:"all_state"`
	Ip                string `help:"IP address" json:"ip"`
}

func (o *MonitorResourceAlertListOptions) GetMasterOpt() string {
	return o.MonitorResourceId
}

func (o *MonitorResourceAlertListOptions) GetSlaveOpt() string {
	return o.AlertId
}

func (o *MonitorResourceAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}
