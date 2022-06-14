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

import "yunion.io/x/onecloud/pkg/apis"

const (
	TapServiceHost  = "host"
	TapServiceGuest = "guest"
)

type NetTapServiceListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
}

type NetTapServiceDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	// 流量镜像目标名称
	Target string `json:"target"`

	// 流量镜像目标IP地址
	TargetIps string `json:"target_ips"`

	// tap flow数量
	FlowCount int `json:"flow_count"`
}

type NetTapServiceCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	// TAP服务类型，监听宿主机的网卡还是虚拟机的网卡, 可能值为 host|guest
	Type string `json:"type" required:"true" choices:"host|guest" help:"type of tap service"`

	// 资源ID，如果Type=host，该值为宿主机的ID，如果Type=guest，该值为虚拟机的ID
	TargetId string `json:"target_id" required:"true" help:"id of target device"`

	// 监听网卡的Mac地址
	MacAddr string `json:"mac_addr" help:"mac address of the device interface for tappping"`
}
