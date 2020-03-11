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

type HostwireDetails struct {
	HostJointResourceDetails

	SHostwire

	// 二层网络名称
	Wire string `json:"wire"`

	// 带宽大小
	Bandwidth int `json:"bandwidth"`
}

type HostwireListInput struct {
	HostJointsListInput
	WireFilterListInput

	// 网桥名称
	Bridge []string `json:"bridge"`

	// 接口名称
	Interface []string `json:"interface"`

	// 是否是主网口
	IsMaster *bool `json:"is_master"`

	// MAC地址
	MacAddr []string `json:"mac_addr"`
}
