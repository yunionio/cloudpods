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

package hcso

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SElbACL struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
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
	acl, err := self.region.GetLoadBalancerAclById(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, acl)
	if err != nil {
		return err
	}

	return nil
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

	params := jsonutils.NewDict()
	whiteListObj := jsonutils.NewDict()
	whiteListObj.Set("whitelist", jsonutils.NewString(whiteList))
	whiteListObj.Set("enable_whitelist", jsonutils.NewBool(true))
	params.Set("whitelist", whiteListObj)
	return DoUpdate(self.region.ecsClient.ElbWhitelist.Update, self.GetId(), params, nil)
}

func (self *SElbACL) Delete() error {
	return DoDelete(self.region.ecsClient.ElbWhitelist.Delete, self.GetId(), nil, nil)
}
