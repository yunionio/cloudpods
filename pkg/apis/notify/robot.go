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

package notify

import "yunion.io/x/onecloud/pkg/apis"

type RobotCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	// description: robot type
	// enum: feishu,dingtalk,workwx,webhook
	// example: webhook
	Type string `json:"type"`
	// description: address
	// example: http://helloworld.io/test/webhook
	Address string `json:"address"`
	// description: Language preference
	// example: zh_CN
	Lang string `json:"lang"`
}

type RobotDetails struct {
	apis.SharableVirtualResourceDetails
}

type RobotListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput
	// description: robot type
	// enum: feishu,dingtalk,workwx,webhook
	// example: webhook
	Type string `json:"type"`
	// description: Language preference
	// example: en
	Lang string `json:"lang"`
}

type RobotUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput
	// description: address
	// example: http://helloworld.io/test/webhook
	Address string `json:"address"`
	// description: Language preference
	// example: en
	Lang string `json:"lang"`
}
