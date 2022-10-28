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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRedisAcl struct {
	redis *SRedisCache
	multicloud.SResourceBase
	AzureTags

	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		Startip string `json:"startIP"`
		Endip   string `json:"endIP"`
	} `json:"properties"`
}

func (self *SRedisAcl) GetId() string {
	return self.ID
}

func (self *SRedisAcl) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SRedisAcl) GetName() string {
	return self.Name
}

func (self *SRedisAcl) GetStatus() string {
	return api.ELASTIC_CACHE_ACL_STATUS_AVAILABLE
}

func (self *SRedisAcl) Refresh() error {
	acl, err := self.redis.region.GetRedisAcl(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, acl)
}

func (self *SRedisAcl) GetIpList() string {
	return fmt.Sprintf("%s-%s", self.Properties.Startip, self.Properties.Endip)
}

func (self *SRedisAcl) Delete() error {
	return self.redis.region.Delete(self.ID)
}

func (self *SRedisAcl) UpdateAcl(securityIps string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "UpdateAcl")
}

func (self *SRegion) GetRedisAcls(id string) ([]SRedisAcl, error) {
	result := struct {
		Value []SRedisAcl
	}{}
	return result.Value, self.get(id+"/firewallRules", url.Values{}, &result)
}

func (self *SRegion) GetRedisAcl(id string) (*SRedisAcl, error) {
	acl := &SRedisAcl{}
	return acl, self.get(id, url.Values{}, acl)
}
