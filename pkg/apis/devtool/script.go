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

package devtool

import "yunion.io/x/onecloud/pkg/apis"

type ScriptApplyInput struct {
	// description: server id
	// required: true
	// example: b48c5c84-9952-4394-8ca9-c3b84e946a03
	ServerID string `json:"server_id"`
}

type ScriptApplyOutput struct {
	// description: Instantiation of script apply
	// example: cf1d1a0f-9b9d-4629-8036-af3ed87c0821
	ScriptApplyId string `json:"script_apply_id"`
}

type ScriptBatchApplyInput struct {
	ServerIds []string `json:"server_ids"`
}

type ScriptBatchApplyOutput struct {
	Results []ScriptBatchApplyResult `json:"results"`
}

type ScriptBatchApplyResult struct {
	ServerId      string `json:"server_id"`
	Succeed       bool   `json:"succeed"`
	Reason        string `json:"reason"`
	ScriptApplyId string `json:"script_apply_id"`
}

type ScriptApplyRecoredListInput struct {
	apis.StatusStandaloneResourceListInput
	// description: Id of Script
	// example: cc2e2ba6-e33d-4be3-8e2d-4d2aa843dd03
	ScriptId string `json:"script_id"`
	// description: Id of Server
	// example:  a4b3n2c9-dbb7-4c51-8e1a-d2d4b331ccec
	ServerId string `json:"server_id"`
	// description: Id of script apply
	// example: a70eb6e6-dbb7-4c51-8e1a-d2d4b331ccec
	ScriptApplyId string `json:"script_apply_id"`
}

type ScriptApplyRecordDetails struct {
	apis.StandaloneResourceDetails
	SScriptApplyRecord
	// description: Id of Script
	// example: cc2e2ba6-e33d-4be3-8e2d-4d2aa843dd03
	ScriptId string `json:"script_id"`
	// description: Id of Server
	// example: a4b3n2c9-dbb7-4c51-8e1a-d2d4b331ccec
	ServerId string `json:"server_id"`
}

type ScriptCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	// description: Id or Name of ansible playbook reference
	// example: cf1d1a0f-9b9d-4629-8036-af3ed87c0821
	PlaybookReference string `json:"playbook_reference"`
	// description: The script may fail to execute, MaxTryTime represents the maximum number of attempts to execute
	MaxTryTimes int `json:"max_try_times"`
}

type ScriptDetails struct {
	apis.SharableVirtualResourceDetails
	SScript
	ApplyInfos []SApplyInfo `json:"apply_infos"`
}

type SApplyInfo struct {
	ServerId string `json:"server_id"`
	TryTimes int    `json:"try_times"`
}

type DevtoolManagerServiceUrlInput struct {
	ServiceName string `json:"service_name"`
	ServerId    string `json:"server_id"`
}

type DevtoolManagerServiceUrlOutput struct {
	ServiceUrl string `json:"service_url"`
}
