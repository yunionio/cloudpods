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

package ucloud

import (
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVipAddr struct {
	VIP      string
	SubnetId string
}

func (ip *SVipAddr) GetIP() string {
	return ip.VIP
}

func (ip *SVipAddr) GetINetworkId() string {
	return ip.SubnetId
}

func (ip *SVipAddr) IsPrimary() bool {
	return true
}

func (ip *SVipAddr) GetGlobalId() string {
	return ip.VIP
}

type SVip struct {
	multicloud.SNetworkInterfaceBase
	region     *SRegion
	CreateTime int64
	Name       string
	RealIp     string
	Remark     string
	SubnetId   string
	Tag        string
	VIP        string
	VIPId      string
	VPCId      string
	Zone       string
}

func (vip *SVip) GetName() string {
	if len(vip.Name) > 0 {
		return vip.Name
	}
	return vip.VIPId
}

func (vip *SVip) GetId() string {
	return vip.VIPId
}

func (vip *SVip) GetGlobalId() string {
	return vip.VIPId
}

func (vip *SVip) GetMacAddress() string {
	ip, _ := netutils.NewIPV4Addr(vip.VIP)
	return ip.ToMac("00:16:")
}

func (vip *SVip) GetAssociateType() string {
	return ""
}

func (vip *SVip) GetAssociateId() string {
	return ""
}

func (vip *SVip) GetStatus() string {
	return api.NETWORK_INTERFACE_STATUS_AVAILABLE
}

func (vip *SVip) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	ip := &SVipAddr{VIP: vip.VIP, SubnetId: vip.SubnetId}
	return []cloudprovider.ICloudInterfaceAddress{ip}, nil
}

func (region *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	vips, err := region.GetVips()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetVips")
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(vips); i++ {
		vips[i].region = region
		ret = append(ret, &vips[i])
	}
	return ret, nil
}

func (self *SRegion) GetVips() ([]SVip, error) {
	vips := []SVip{}
	params := NewUcloudParams()

	err := self.DoListAll("DescribeVIP", params, &vips)
	return vips, err
}
