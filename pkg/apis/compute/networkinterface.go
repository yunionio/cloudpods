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

type NetworkInterfaceNetworkInfo struct {
	// Ip子网Id
	NetworkId string `json:"network_id"`
	// IP地址
	IpAddr string `json:"ip_addr"`
	// 是否是主ip地址
	Primary bool `json:"primary"`
	// 弹性网卡id
	NetworkinterfaceId string `json:"networkinterface_id"`
	// IP子网名称
	Network string `json:"network"`
}

type NetworkInterfaceDetails struct {
	apis.StatusInfrasResourceBaseDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SNetworkInterface

	// 弹性网卡网络信息
	Networks []NetworkInterfaceNetworkInfo `json:"networks"`
}
