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

package compute

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	CloudaccountSyncTaskKeyword       = "cloudaccountsynctask"
	CloudaccountSyncTaskKeywordPlural = "cloudaccountsynctasks"

	CloudaccountSyncTaskLevelAccount  = "account"
	CloudaccountSyncTaskLevelProvider = "provider"
	CloudaccountSyncTaskLevelRegion   = "region"

	CloudaccountSyncTaskStatusQueued  = "queued"
	CloudaccountSyncTaskStatusRunning = "running"
	CloudaccountSyncTaskStatusDone    = "done"
	CloudaccountSyncTaskStatusFail    = "fail"
	CloudaccountSyncTaskStatusCancel  = "cancel"

	CloudaccountSyncTaskModeAuto   = "auto"
	CloudaccountSyncTaskModeManual = "manual"
)

var CloudaccountSyncTaskStatusActive = []string{
	CloudaccountSyncTaskStatusQueued,
	CloudaccountSyncTaskStatusRunning,
}

var CloudaccountSyncTaskStatusTerminal = []string{
	CloudaccountSyncTaskStatusDone,
	CloudaccountSyncTaskStatusFail,
	CloudaccountSyncTaskStatusCancel,
}

type CloudaccountSyncTaskListInput struct {
	apis.DomainLevelResourceListInput
	apis.StatusResourceBaseListInput

	// 云账号 ID
	CloudaccountId string `json:"cloudaccount_id"`
	// 云订阅 ID
	CloudproviderId string `json:"cloudprovider_id"`
	// 区域 ID
	CloudregionId string `json:"cloudregion_id"`
	// 父任务 ID
	ParentId string `json:"parent_id"`
	// 任务层级: account / provider / region
	Level string `json:"level"`
	// 仅展示账号级任务（列表默认）
	AccountLevel bool `json:"account_level"`
	// 展示全部状态
	All bool `json:"all"`
	// 仅展示进行中
	Active bool `json:"active"`
	// 仅展示运行中
	Running bool `json:"running"`
}

type CloudaccountSyncTaskDetails struct {
	apis.DomainLevelResourceDetails

	Status string `json:"status"`

	Cloudaccount  string               `json:"cloudaccount"`
	Cloudprovider string               `json:"cloudprovider"`
	Cloudregion   string               `json:"cloudregion"`
	ParentId      string               `json:"parent_id"`
	Level         string               `json:"level"`
	Mode          string               `json:"mode"`
	QueuedAt      time.Time            `json:"queued_at"`
	StartedAt     time.Time            `json:"started_at"`
	EndedAt       time.Time            `json:"ended_at"`
	ChildTotal    int                  `json:"child_total"`
	ChildDone     int                  `json:"child_done"`
	ChildFail     int                  `json:"child_fail"`
	ChildCancel   int                  `json:"child_cancel"`
	ChildRunning  int                  `json:"child_running"`
	ChildQueued   int                  `json:"child_queued"`
	SyncResults   jsonutils.JSONObject `json:"sync_results"`
}

type CloudaccountSyncTaskCancelInput struct {
	Reason string `json:"reason"`
}
