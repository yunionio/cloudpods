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

package cloudevent

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

type CloudeventListInput struct {
	apis.ModelBaseListInput

	compute.CloudenvResourceListInput

	// 服务类型
	Service []string `json:"service"`

	// 订阅
	Manager []string `json:"manager"`

	// 账号
	Account []string `json:"account"`

	// 操作类型
	Action []string `json:"action"`

	// 操作日志起始时间
	Since time.Time `json:"since"`
	// 操作日志截止时间
	Until time.Time `json:"until"`
}
