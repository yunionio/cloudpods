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

package hcs

import (
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElbACL struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	region *SRegion

	Id              string `json:"id"`
	ListenerId      string `json:"listener_id"`
	TenantId        string `json:"tenant_id"`
	EnableWhitelist bool   `json:"enable_whitelist"`
	Whitelist       string `json:"whitelist"`
}

func (self *SElbACL) GetAclListenerID() string {
	return self.ListenerId
}

func (self *SElbACL) GetId() string {
	return self.Id
}

func (self *SElbACL) GetName() string {
	return self.Id
}

func (self *SElbACL) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbACL) GetStatus() string {
	if self.EnableWhitelist {
		return api.LB_BOOL_ON
	}

	return api.LB_BOOL_OFF
}

func (self *SElbACL) Refresh() error {
	acl, err := self.region.GetLoadBalancerAcl(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, acl)
}

func (self *SElbACL) IsEmulated() bool {
	return false
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
		"enable_whitelist": acl.AccessControlEnable,
	}
	return self.region.lbUpdate("lbaas/whitelists/"+self.GetId(), map[string]interface{}{"whitelist": params})
}

func (self *SElbACL) Delete() error {
	return self.region.lbDelete("lbaas/whitelists/" + self.GetId())
}

func (self *SRegion) GetLoadBalancerAcl(aclId string) (*SElbACL, error) {
	ret := &SElbACL{region: self}
	return ret, self.lbGet("lbaas/whitelists/"+aclId, ret)
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561582.html
func (self *SRegion) GetLoadBalancerAcls(listenerId string) ([]SElbACL, error) {
	query := url.Values{}
	if len(listenerId) > 0 {
		query.Set("listener_id", listenerId)
	}
	ret := []SElbACL{}
	return ret, self.lbList("lbaas/whitelists", query, &ret)
}

func (self *SRegion) CreateLoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (*SElbACL, error) {
	params := map[string]interface{}{
		"listener_id": acl.ListenerId,
	}
	if len(acl.Entrys) > 0 {
		whitelist := []string{}
		for i := range acl.Entrys {
			whitelist = append(whitelist, acl.Entrys[i].CIDR)
		}
		params["enable_whitelist"] = acl.AccessControlEnable
		params["whitelist"] = strings.Join(whitelist, ",")
	} else {
		params["enable_whitelist"] = false
	}
	ret := &SElbACL{region: self}
	return ret, self.lbCreate("lbaas/whitelists", map[string]interface{}{"whitelist": params}, ret)
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	ret, err := self.CreateLoadBalancerAcl(acl)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	acl, err := self.GetLoadBalancerAcl(aclId)
	if err != nil {
		return nil, err
	}
	return acl, nil
}
