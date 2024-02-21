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

package huawei

import (
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbACL struct {
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion

	ID              string `json:"id"`
	ListenerID      string `json:"listener_id"`
	TenantID        string `json:"tenant_id"`
	EnableWhitelist bool   `json:"enable_whitelist"`
	Whitelist       string `json:"whitelist"`
}

func (self *SElbACL) GetId() string {
	return self.ID
}

func (self *SElbACL) GetName() string {
	return self.ID
}

func (self *SElbACL) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbACL) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (self *SElbACL) Refresh() error {
	acl, err := self.region.GetLoadBalancerAcl(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, acl)
}

func (self *SElbACL) GetProjectId() string {
	return ""
}

func (self *SElbACL) GetAclEntries() []cloudprovider.SLoadbalancerAccessControlListEntry {
	ret := []cloudprovider.SLoadbalancerAccessControlListEntry{}
	for _, cidr := range strings.Split(self.Whitelist, ",") {
		ret = append(ret, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: cidr})
	}

	return ret
}

func (self *SElbACL) Sync(acl *cloudprovider.SLoadbalancerAccessControlList) error {
	whiteList := ""
	cidrs := []string{}
	for _, entry := range acl.Entrys {
		cidrs = append(cidrs, entry.CIDR)
	}
	whiteList = strings.Join(cidrs, ",")
	params := map[string]interface{}{
		"whitelist":        whiteList,
		"enable_whitelist": true,
	}
	_, err := self.region.put(SERVICE_ELB, "elb/whitelists/"+self.GetId(), map[string]interface{}{"whitelist": params})
	return err
}

func (self *SElbACL) Delete() error {
	_, err := self.region.delete(SERVICE_ELB, "elb/whitelists/"+self.GetId())
	return err
}

func (self *SRegion) GetLoadBalancerAcl(aclId string) (*SElbACL, error) {
	resp, err := self.list(SERVICE_ELB, "elb/whitelists/"+aclId, nil)
	if err != nil {
		return nil, err
	}
	ret := &SElbACL{region: self}
	return ret, resp.Unmarshal(ret, "whitelist")
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ELB/doc?version=v2&api=ListWhitelists
func (self *SRegion) GetLoadBalancerAcls(listenerId string) ([]SElbACL, error) {
	query := url.Values{}
	if len(listenerId) > 0 {
		query.Set("listener_id", listenerId)
	}
	resp, err := self.list(SERVICE_ELB, "elb/whitelists", query)
	if err != nil {
		return nil, err
	}
	ret := []SElbACL{}
	return ret, resp.Unmarshal(&ret, "whitelists")
}

func (self *SRegion) CreateLoadBalancerAcl(listenerId string, opts *cloudprovider.SLoadbalancerAccessControlList) (*SElbACL, error) {
	params := map[string]interface{}{
		"listener_id": listenerId,
	}
	if len(opts.Entrys) > 0 {
		whitelist := []string{}
		for i := range opts.Entrys {
			whitelist = append(whitelist, opts.Entrys[i].CIDR)
		}
		params["enable_whitelist"] = "true"
		params["whitelist"] = strings.Join(whitelist, ",")
	} else {
		params["enable_whitelist"] = false
	}
	resp, err := self.post(SERVICE_ELB, "elb/whitelists", map[string]interface{}{"whitelist": params})
	if err != nil {
		return nil, err
	}
	ret := &SElbACL{region: self}
	return ret, resp.Unmarshal(ret, "whitelist")
}
