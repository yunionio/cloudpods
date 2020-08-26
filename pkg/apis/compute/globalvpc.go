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
	"yunion.io/x/onecloud/pkg/apis"
)

type GlobalVpcCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput
}

type GlobalVpcDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails

	SGlobalVpc

	// vpc数量
	VpcCount int `json:"vpc_count"`
}

type GlobalVpcResourceInfo struct {
	// 全局VPC名称
	Globalvpc string `json:"globalvpc"`
}

type GlobalVpcResourceInput struct {
	// GlobalVpc ID or Name
	GlobalvpcId string `json:"globalvpc_id"`

	// swagger:ignore
	// Deprecated
	Globalvpc string `json:"globalvpc" yunion-deprecated-by:"globalvpc_id"`
}

type GlobalVpcResourceListInput struct {
	GlobalVpcResourceInput

	// 以GlobalVpc的名称排序
	OrderByGlobalvpc string `json:"order_by_globalvpc"`
}

type GlobalvpcUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}

type GlobalVpcListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
}
