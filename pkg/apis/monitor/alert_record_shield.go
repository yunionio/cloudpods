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

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertRecordShieldCreateInput struct {
	apis.StandaloneResourceCreateInput

	AlertId string `json:"alert_id"`
	ResType string `json:"res_type"`
	ResName string `json:"res_name"`
	ResId   string `json:"res_id"`

	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type AlertRecordShieldDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	CommonAlertDetails
	AlertName string `json:"alert_name"`
	ResName   string `json:"res_name"`
	Expired   bool   `json:"expired"`
}

type AlertRecordShieldListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertName string     `json:"alert_name"`
	ResType   string     `json:"res_type"`
	ResName   string     `json:"res_name"`
	ResId     string     `json:"res_id"`
	AlertId   string     `json:"alert_id"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}
