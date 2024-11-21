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

package baidu

import "yunion.io/x/cloudmux/pkg/cloudprovider"

type Ips struct {
	PrivateIP       string
	Eip             string
	Primary         bool
	EipId           string
	EipAllocationId string
	EipSize         string
	EipStatus       string
	EipGroupId      string
	EipType         string
}

type NicInfo struct {
	cloudprovider.DummyICloudNic

	EniId          string
	EniUUId        string
	Name           string
	Type           string
	SubnetId       string
	SubnetType     string
	Az             string
	Description    string
	DeviceId       string
	Status         string
	MacAddress     string
	VpcId          string
	CreatedTime    string
	EniNum         int
	EriNum         int
	Ips            []Ips
	SecurityGroups []string
}

func (nic *NicInfo) GetId() string {
	return nic.EniId
}

func (nic *NicInfo) GetIP() string {
	for _, ip := range nic.Ips {
		if ip.Primary {
			return ip.PrivateIP
		}
	}
	return ""
}

func (nic *NicInfo) InClassicNetwork() bool {
	return false
}

func (nic *NicInfo) GetIP6() string {
	return ""
}

func (nic *NicInfo) GetDriver() string {
	return "virtio"
}

func (nic *NicInfo) GetINetworkId() string {
	return nic.SubnetId
}

func (nic *NicInfo) GetMAC() string {
	return nic.MacAddress
}

func (nic *NicInfo) GetSubAddress() ([]string, error) {
	ret := []string{}
	for _, ip := range nic.Ips {
		if ip.Primary {
			continue
		}
		if len(ip.PrivateIP) > 0 {
			ret = append(ret, ip.PrivateIP)
		}
	}
	return ret, nil
}
