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
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

const (
	MONITOR_RESOURCE_ALERT_STATUS_INIT     = "init"
	MONITOR_RESOURCE_ALERT_STATUS_ATTACH   = "attach"
	MONITOR_RESOURCE_ALERT_STATUS_ALERTING = "alerting"
)

type MonitorResourceCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	ResId       string `json:"res_id"`
	ResType     string `json:"res_type"`
	AlertStatus string `json:"alert_status"`
}

type MonitorResourceListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
	compute.ManagedResourceListInput

	ResId     []string `json:"res_id"`
	ResType   string   `json:"res_type"`
	OnlyResId bool     `json:"only_res_id"`
}

type MonitorResourceDetails struct {
	apis.VirtualResourceDetails
	compute.CloudregionResourceInfo

	AttachAlertCount int64 `json:"attach_alert_count"`
}
