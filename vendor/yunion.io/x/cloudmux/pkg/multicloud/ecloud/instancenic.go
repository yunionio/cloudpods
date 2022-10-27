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

	Id               string
	PrivateIp        string
	FipAddress       string
	FipBandwidthSize int
	PortId           string
	PortName         string
	FixedIpDetails   []SFixedIpDetail
	RouterId         string
}

type SInstanceNicDetail struct {
	MacAddress     string
	SecurityGroups []SSecurityGroupRef
	Status         int
	ResourceId     string
	CreaetTime     time.Time
	PublicIp       string
	IpId           string
	NetworkId      string
}

type SSecurityGroupRef struct {
	Id   string
	Name string
}

type SFixedIpDetail struct {
	IpAddress     string
	IpVersion     string
	PublicIp      string
	BandWidthSize int
	BandWidthType string
	SubnetId      string
	SubnetName    string
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
