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
	ServerID string
	// description: whether to use eip first
	// example: true
	EipFirst bool
	// description: Id of proxyEndpoint
	// example: cf1d1a0f-9b9d-4629-8036-af3ed87c0821
	ProxyEndpointId string
	// description: whether to automatically select proxy endpoint
	AutoChooseProxyEndpoint bool
}

type ScriptApplyOutput struct {
	// description: Instantiation of script apply
	// example: cf1d1a0f-9b9d-4629-8036-af3ed87c0821
	ScriptApplyId string
}

type ScriptApplyRecoredListInput struct {
	apis.StatusStandaloneResourceListInput
	// description: Id of Script
	// example: cc2e2ba6-e33d-4be3-8e2d-4d2aa843dd03
	ScriptId string
}

type ScriptCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	// description: Id or Name of ansible playbook reference
	// example: cf1d1a0f-9b9d-4629-8036-af3ed87c0821
	PlaybookReference string
	// description: The script may fail to execute, MaxTryTime represents the maximum number of attempts to execute
	MaxTryTimes int
}

type ScriptDetails struct {
	apis.SharableVirtualResourceDetails
	SScript
	ApplyInfos []SApplyInfo
}

type SApplyInfo struct {
	ServerId        string
	EipFirst        bool
	ProxyEndpointId string
	TryTimes        int
}
