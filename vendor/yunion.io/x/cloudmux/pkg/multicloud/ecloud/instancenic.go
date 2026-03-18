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

package ecloud

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	cloudprovider.DummyICloudNic
	SInstanceNicDetail

	instance *SInstance

	Id               string         `json:"id,omitempty"` // 网卡ID（等同 PortId）
	PrivateIp        string         `json:"-"`
	FipAddress       string         `json:"-"`
	FipBandwidthSize int            `json:"-"`
	PortId           string         `json:"portId,omitempty"`   // 兼容旧字段
	PortName         string         `json:"portName,omitempty"` // 兼容旧字段
	FixedIpDetails   []SFixedIpDetail `json:"-"`
	RouterId         string         `json:"routerId,omitempty"`
}

type SInstanceNicDetail struct {
	MacAddress     string             `json:"macAddress,omitempty"`
	SecurityGroups []SSecurityGroupRef `json:"securityGroups,omitempty"`
	Status         int                `json:"status,omitempty"`
	ResourceId     string             `json:"resourceId,omitempty"`

	// CreateTime 正确拼写；保留旧字段 CreaetTime 以兼容历史引用
	CreateTime time.Time `json:"createTime,omitempty"`
	CreaetTime time.Time `json:"-"`

	PublicIp  string `json:"publicIp,omitempty"`
	IpId      string `json:"ipId,omitempty"`
	NetworkId string `json:"networkId,omitempty"`
}

type SSecurityGroupRef struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type SFixedIpDetail struct {
	IpAddress     string `json:"ipAddress,omitempty"`
	IpVersion     string `json:"ipVersion,omitempty"`
	PublicIp      string `json:"publicIp,omitempty"`
	BandWidthSize int    `json:"bandWidthSize,omitempty"`
	BandWidthType string `json:"bandWidthType,omitempty"`
	SubnetId      string `json:"subnetId,omitempty"`
	SubnetName    string `json:"subnetName,omitempty"`
}

func (in *SInstanceNic) GetIP() string {
	return in.PrivateIp
}

func (in *SInstanceNic) GetMAC() string {
	return in.MacAddress
}

func (in *SInstanceNic) GetId() string {
	return in.PortId
}

func (in *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (in *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (in *SInstanceNic) GetINetworkId() string {
	return in.NetworkId
}
