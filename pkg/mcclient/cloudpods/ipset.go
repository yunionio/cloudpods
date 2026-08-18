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

package cloudpods

import (
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SIpSet struct {
	multicloud.SVirtualResourceBase
	CloudpodsTags
	region *SRegion

	api.IpSetDetails
}

func (self *SIpSet) GetName() string {
	return self.Name
}

func (self *SIpSet) GetId() string {
	return self.Id
}

func (self *SIpSet) GetGlobalId() string {
	return self.Id
}

func (self *SIpSet) GetStatus() string {
	return self.Status
}

func (self *SIpSet) GetProjectId() string {
	return self.TenantId
}

func (self *SIpSet) GetDescription() string {
	return self.Description
}

func (self *SIpSet) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SIpSet) Refresh() error {
	ipSet, err := self.region.GetIpSet(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ipSet)
}

func (self *SIpSet) GetIpSetType() string {
	return string(self.IpSetType)
}

func (self *SIpSet) GetAddresses() []string {
	if len(self.Data) == 0 {
		return []string{}
	}
	parts := strings.Split(self.Data, ",")
	ret := make([]string, 0, len(parts))
	for i := range parts {
		addr := strings.TrimSpace(parts[i])
		if len(addr) > 0 {
			ret = append(ret, addr)
		}
	}
	return ret
}

func (self *SIpSet) Update(opts *cloudprovider.IpSetUpdateOptions) error {
	input := api.IpSetUpdateInput{}
	if len(opts.Name) > 0 {
		input.Name = opts.Name
	}
	if len(opts.Addresses) > 0 {
		data := strings.Join(opts.Addresses, ",")
		input.Data = &data
	}
	return self.region.cli.update(&modules.IpSets, self.Id, input)
}

func (self *SIpSet) Delete() error {
	return self.region.cli.delete(&modules.IpSets, self.Id)
}

func (self *SRegion) GetIpSets() ([]SIpSet, error) {
	ret := []SIpSet{}
	return ret, self.list(&modules.IpSets, map[string]interface{}{"cloud_env": "onpremise"}, &ret)
}

func (self *SRegion) GetIpSet(id string) (*SIpSet, error) {
	ipSet := &SIpSet{region: self}
	return ipSet, self.cli.get(&modules.IpSets, id, nil, ipSet)
}

func (self *SRegion) GetIIpSets() ([]cloudprovider.ICloudIpSet, error) {
	ipSets, err := self.GetIpSets()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudIpSet{}
	for i := range ipSets {
		ipSets[i].region = self
		ret = append(ret, &ipSets[i])
	}
	return ret, nil
}

func (self *SRegion) GetIIpSetById(id string) (cloudprovider.ICloudIpSet, error) {
	ipSet, err := self.GetIpSet(id)
	if err != nil {
		return nil, err
	}
	return ipSet, nil
}

func (self *SRegion) CreateIIpSet(opts *cloudprovider.IpSetCreateOptions) (cloudprovider.ICloudIpSet, error) {
	ipSetType := opts.IpSetType
	if len(ipSetType) == 0 {
		ipSetType = cloudprovider.IpSetTypeIpv4CidrList
		for _, addr := range opts.Addresses {
			if strings.Contains(addr, ":") {
				ipSetType = cloudprovider.IpSetTypeIpv6CidrList
				break
			}
		}
	}
	input := api.IpSetCreateInput{}
	input.Name = opts.Name
	input.Description = opts.Desc
	input.IpSetType = api.TIpSetType(ipSetType)
	input.Data = strings.Join(opts.Addresses, ",")
	input.CloudregionId = self.Id
	ipSet := &SIpSet{region: self}
	return ipSet, self.create(&modules.IpSets, input, ipSet)
}
